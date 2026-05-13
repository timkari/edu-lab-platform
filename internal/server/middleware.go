package server

import (
	"encoding/json"
	"net/http"

	"github.com/edu-lab-platform/internal/auth"
	"github.com/edu-lab-platform/internal/db"
	"github.com/edu-lab-platform/internal/models"
)

func withBearer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := auth.ParseToken(r.Header.Get("Authorization"))
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, Response{OK: false, Error: "unauthorized"})
			return
		}
		next.ServeHTTP(w, r.WithContext(auth.WithClaims(r.Context(), c)))
	})
}

func requireRoles(roles ...string) func(http.Handler) http.Handler {
	set := map[string]struct{}{}
	for _, role := range roles {
		set[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := auth.RoleFromContext(r.Context())
			if _, ok := set[role]; !ok {
				writeJSON(w, http.StatusForbidden, Response{OK: false, Error: "forbidden"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func chainAdmin(next http.Handler) http.Handler {
	if db.DB() == nil {
		return next
	}
	return withBearer(requireRoles("admin")(next))
}

func chainStudent(next http.Handler) http.Handler {
	if db.DB() == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusServiceUnavailable, Response{OK: false, Error: "база данных не настроена"})
		})
	}
	return withBearer(requireRoles("student")(next))
}

func chainStudentOrAdmin(next http.Handler) http.Handler {
	if db.DB() == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusServiceUnavailable, Response{OK: false, Error: "база данных не настроена"})
		})
	}
	return withBearer(requireRoles("student", "admin")(next))
}

// POST /api/auth/login — без JWT; тело: { "student_id", "password" }
func handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{OK: false, Error: "POST only"})
		return
	}
	if db.DB() == nil {
		writeJSON(w, http.StatusServiceUnavailable, Response{OK: false, Error: "база данных не настроена"})
		return
	}
	var body struct {
		StudentID string `json:"student_id"`
		Password  string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	u, err := auth.Login(body.StudentID, body.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, Response{OK: false, Error: "неверный логин или пароль"})
		return
	}
	token, err := auth.IssueToken(u)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: map[string]any{
		"token":      token,
		"student_id": u.StudentID,
		"role":       u.Role,
	}})
}

// GET /api/templates — список шаблонов (студент или админ, с JWT)
func handleTemplatesList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{OK: false, Error: "GET only"})
		return
	}
	g := db.DB()
	if g == nil {
		writeJSON(w, http.StatusServiceUnavailable, Response{OK: false, Error: "база данных не настроена"})
		return
	}
	var list []models.Template
	if err := g.Order("id").Find(&list).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: list})
}
