// Обработчики заявок на ВМ: POST/GET /api/request/* и GET /api/session/me.
// Требуются PostgreSQL (DATABASE_URL), JWT после входа через POST /api/auth/login.
// Поведение и примеры тел запросов описаны в README.md.
package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/edu-lab-platform/internal/auth"
	"github.com/edu-lab-platform/internal/backup"
	"github.com/edu-lab-platform/internal/db"
	"github.com/edu-lab-platform/internal/lab"
	"github.com/edu-lab-platform/internal/models"
	"github.com/edu-lab-platform/internal/store"
	"gorm.io/gorm"
)

// POST /api/request/create (student)
func handleRequestCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{OK: false, Error: "POST only"})
		return
	}
	g := db.DB()
	if g == nil {
		writeJSON(w, http.StatusServiceUnavailable, Response{OK: false, Error: "база данных не настроена"})
		return
	}
	studentID := auth.StudentIDFromContext(r.Context())
	var body struct {
		TemplateID  uint   `json:"template_id"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	if body.TemplateID == 0 || body.Description == "" {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: "template_id и description обязательны"})
		return
	}
	active, err := store.HasActiveVM(studentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	if active {
		writeJSON(w, http.StatusConflict, Response{OK: false, Error: "У вас уже есть активная ВМ"})
		return
	}
	pending, err := store.PendingCreateRequestExists(studentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	if pending {
		writeJSON(w, http.StatusConflict, Response{OK: false, Error: "Уже есть ожидающая заявка на создание ВМ"})
		return
	}
	var tpl models.Template
	if err := g.First(&tpl, body.TemplateID).Error; err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: "шаблон не найден"})
		return
	}
	req := models.Request{
		StudentID:   studentID,
		TemplateID:  tpl.ID,
		Description: body.Description,
		Type:        models.RequestTypeCreate,
		Status:      models.RequestStatusPending,
	}
	if err := g.Create(&req).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: map[string]uint{"request_id": req.ID}})
}

// POST /api/request/delete (student)
func handleRequestDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{OK: false, Error: "POST only"})
		return
	}
	g := db.DB()
	if g == nil {
		writeJSON(w, http.StatusServiceUnavailable, Response{OK: false, Error: "база данных не настроена"})
		return
	}
	studentID := auth.StudentIDFromContext(r.Context())
	var body struct {
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	if body.Description == "" {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: "description обязателен"})
		return
	}
	sess, err := store.GetActiveSession(studentID)
	if err != nil || sess == nil {
		writeJSON(w, http.StatusConflict, Response{OK: false, Error: "нет активной ВМ для удаления"})
		return
	}
	req := models.Request{
		StudentID:   studentID,
		TemplateID:  sess.TemplateID,
		Description: body.Description,
		Type:        models.RequestTypeDelete,
		Status:      models.RequestStatusPending,
	}
	if err := g.Create(&req).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: map[string]uint{"request_id": req.ID}})
}

// GET /api/request/my (student)
func handleRequestMy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{OK: false, Error: "GET only"})
		return
	}
	g := db.DB()
	if g == nil {
		writeJSON(w, http.StatusServiceUnavailable, Response{OK: false, Error: "база данных не настроена"})
		return
	}
	studentID := auth.StudentIDFromContext(r.Context())
	var list []models.Request
	if err := g.Where("student_id = ?", studentID).Order("created_at desc").Find(&list).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	out := make([]map[string]any, 0, len(list))
	for i := range list {
		out = append(out, requestToDTO(g, &list[i]))
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: out})
}

func requestToDTO(g *gorm.DB, req *models.Request) map[string]any {
	var tpl models.Template
	_ = g.First(&tpl, req.TemplateID).Error
	row := map[string]any{
		"id":              req.ID,
		"student_id":      req.StudentID,
		"template_id":     req.TemplateID,
		"template_name":   tpl.Name,
		"description":     req.Description,
		"type":            req.Type,
		"status":          req.Status,
		"admin_comment":   req.AdminComment,
		"created_at":      req.CreatedAt.Format(time.RFC3339),
		"updated_at":      req.UpdatedAt.Format(time.RFC3339),
		"vm_url":          nil,
		"session_running": false,
	}
	if req.Type == models.RequestTypeCreate && req.Status == models.RequestStatusApproved {
		var s models.Session
		if err := g.Where("request_id = ?", req.ID).First(&s).Error; err == nil {
			run, _ := lab.IsRunning(req.StudentID)
			if s.Status == models.SessionRunning && run {
				url, _ := lab.Info(req.StudentID)
				row["vm_url"] = url
				row["session_running"] = true
			} else if s.AccessURL != "" {
				row["vm_url"] = s.AccessURL
			}
		}
	}
	return row
}

// GET /api/request/all (admin) — query: status, type
func handleRequestAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{OK: false, Error: "GET only"})
		return
	}
	g := db.DB()
	if g == nil {
		writeJSON(w, http.StatusServiceUnavailable, Response{OK: false, Error: "база данных не настроена"})
		return
	}
	q := g.Model(&models.Request{}).Order("created_at desc")
	if st := r.URL.Query().Get("status"); st != "" {
		q = q.Where("status = ?", st)
	}
	if tp := r.URL.Query().Get("type"); tp != "" {
		q = q.Where("type = ?", tp)
	}
	var list []models.Request
	if err := q.Find(&list).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	out := make([]map[string]any, 0, len(list))
	for i := range list {
		out = append(out, requestToDTO(g, &list[i]))
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: out})
}

// POST /api/request/approve (admin)
func handleRequestApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{OK: false, Error: "POST only"})
		return
	}
	g := db.DB()
	if g == nil {
		writeJSON(w, http.StatusServiceUnavailable, Response{OK: false, Error: "база данных не настроена"})
		return
	}
	var body struct {
		RequestID uint `json:"request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	if body.RequestID == 0 {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: "request_id обязателен"})
		return
	}
	var req models.Request
	if err := g.First(&req, body.RequestID).Error; err != nil {
		writeJSON(w, http.StatusNotFound, Response{OK: false, Error: "заявка не найдена"})
		return
	}
	if req.Status != models.RequestStatusPending {
		writeJSON(w, http.StatusConflict, Response{OK: false, Error: "заявка уже обработана"})
		return
	}
	base := BasePath()

	switch req.Type {
	case models.RequestTypeCreate:
		active, err := store.HasActiveVM(req.StudentID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
			return
		}
		if active {
			req.Status = models.RequestStatusRejected
			req.AdminComment = "У студента уже появилась активная ВМ"
			_ = g.Save(&req).Error
			writeJSON(w, http.StatusConflict, Response{OK: false, Error: req.AdminComment})
			return
		}
		var tpl models.Template
		if err := g.First(&tpl, req.TemplateID).Error; err != nil {
			writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: "шаблон не найден"})
			return
		}
		if err := backup.EnsureStructure(base); err != nil {
			writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
			return
		}
		if err := lab.Start(base, req.StudentID, tpl.DockerImage); err != nil {
			writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
			return
		}
		url, pwd := lab.Info(req.StudentID)
		workDir, _ := lab.WorkDirPath(base, req.StudentID)
		rid := req.ID
		sess := models.Session{
			StudentID:   req.StudentID,
			TemplateID:  req.TemplateID,
			RequestID:   &rid,
			Status:      models.SessionRunning,
			AccessURL:   url,
			LastSeenAt:  time.Now(),
		}
		if err := g.Create(&sess).Error; err != nil {
			_ = lab.Stop(req.StudentID)
			writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
			return
		}
		req.Status = models.RequestStatusApproved
		if err := g.Save(&req).Error; err != nil {
			writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, Response{OK: true, Data: map[string]any{
			"session_id": sess.ID,
			"url":        url,
			"password":   pwd,
			"work_dir":   workDir,
		}})
		return

	case models.RequestTypeDelete:
		sess, err := store.GetActiveSession(req.StudentID)
		if err != nil || sess == nil {
			req.Status = models.RequestStatusRejected
			req.AdminComment = "Активная сессия не найдена"
			_ = g.Save(&req).Error
			writeJSON(w, http.StatusConflict, Response{OK: false, Error: req.AdminComment})
			return
		}
		_, _ = backup.Create(base, req.StudentID)
		_ = lab.Stop(req.StudentID)
		sess.Status = models.SessionTerminated
		if err := g.Save(sess).Error; err != nil {
			writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
			return
		}
		req.Status = models.RequestStatusApproved
		if err := g.Save(&req).Error; err != nil {
			writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, Response{OK: true})
		return

	default:
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: "неизвестный тип заявки"})
	}
}

// POST /api/request/reject (admin)
func handleRequestReject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{OK: false, Error: "POST only"})
		return
	}
	g := db.DB()
	if g == nil {
		writeJSON(w, http.StatusServiceUnavailable, Response{OK: false, Error: "база данных не настроена"})
		return
	}
	var body struct {
		RequestID uint   `json:"request_id"`
		Comment   string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	if body.RequestID == 0 {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: "request_id обязателен"})
		return
	}
	var req models.Request
	if err := g.First(&req, body.RequestID).Error; err != nil {
		writeJSON(w, http.StatusNotFound, Response{OK: false, Error: "заявка не найдена"})
		return
	}
	if req.Status != models.RequestStatusPending {
		writeJSON(w, http.StatusConflict, Response{OK: false, Error: "заявка уже обработана"})
		return
	}
	req.Status = models.RequestStatusRejected
	req.AdminComment = body.Comment
	if err := g.Save(&req).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true})
}

// POST /api/request/cancel (student)
func handleRequestCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{OK: false, Error: "POST only"})
		return
	}
	g := db.DB()
	if g == nil {
		writeJSON(w, http.StatusServiceUnavailable, Response{OK: false, Error: "база данных не настроена"})
		return
	}
	studentID := auth.StudentIDFromContext(r.Context())
	var body struct {
		RequestID uint `json:"request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	if body.RequestID == 0 {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: "request_id обязателен"})
		return
	}
	var req models.Request
	if err := g.First(&req, body.RequestID).Error; err != nil {
		writeJSON(w, http.StatusNotFound, Response{OK: false, Error: "заявка не найдена"})
		return
	}
	if req.StudentID != studentID {
		writeJSON(w, http.StatusForbidden, Response{OK: false, Error: "чужая заявка"})
		return
	}
	if req.Status != models.RequestStatusPending {
		writeJSON(w, http.StatusConflict, Response{OK: false, Error: "можно отменить только ожидающую заявку"})
		return
	}
	req.Status = models.RequestStatusCancelled
	if err := g.Save(&req).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true})
}

// POST /api/auth/logout (student или admin) — у студента с активной ВМ: бэкап, docker stop, сессия terminated.
func handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{OK: false, Error: "POST only"})
		return
	}
	g := db.DB()
	if g == nil {
		writeJSON(w, http.StatusServiceUnavailable, Response{OK: false, Error: "база данных не настроена"})
		return
	}
	studentID := auth.StudentIDFromContext(r.Context())
	role := auth.RoleFromContext(r.Context())

	if role == "student" {
		sess, err := store.GetActiveSession(studentID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
			return
		}
		base := BasePath()
		if sess != nil {
			_, _ = backup.Create(base, studentID)
		}
		_ = lab.Stop(studentID)
		if sess != nil {
			sess.Status = models.SessionTerminated
			if err := g.Save(sess).Error; err != nil {
				writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
				return
			}
		}
	}
	writeJSON(w, http.StatusOK, Response{OK: true})
}

// POST /api/session/ping (student) — отметка «студент в сети» для тайм-аута бездействия.
func handleSessionPing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{OK: false, Error: "POST only"})
		return
	}
	g := db.DB()
	if g == nil {
		writeJSON(w, http.StatusServiceUnavailable, Response{OK: false, Error: "база данных не настроена"})
		return
	}
	studentID := auth.StudentIDFromContext(r.Context())
	sess, err := store.GetActiveSession(studentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	if sess == nil {
		writeJSON(w, http.StatusOK, Response{OK: true, Data: map[string]any{"updated": false}})
		return
	}
	sess.LastSeenAt = time.Now()
	if err := g.Save(sess).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: map[string]any{"updated": true}})
}

// GET /api/session/me (student) — активная сессия
func handleSessionMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{OK: false, Error: "GET only"})
		return
	}
	g := db.DB()
	if g == nil {
		writeJSON(w, http.StatusServiceUnavailable, Response{OK: false, Error: "база данных не настроена"})
		return
	}
	studentID := auth.StudentIDFromContext(r.Context())
	sess, err := store.GetActiveSession(studentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	if sess == nil {
		writeJSON(w, http.StatusOK, Response{OK: true, Data: map[string]any{"active": false}})
		return
	}
	run, _ := lab.IsRunning(studentID)
	url := sess.AccessURL
	pwd := ""
	if run {
		url, pwd = lab.Info(studentID)
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: map[string]any{
		"active":            true,
		"session_id":        sess.ID,
		"status":            sess.Status,
		"container_running": run,
		"url":               url,
		"password":          pwd,
		"template_id":       sess.TemplateID,
	}})
}
