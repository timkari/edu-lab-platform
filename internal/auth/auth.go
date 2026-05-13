package auth

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/edu-lab-platform/internal/db"
	"github.com/edu-lab-platform/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type ctxKey string

const (
	ctxStudentID ctxKey = "student_id"
	ctxRole      ctxKey = "role"
)

// Claims embedded in JWT.
type Claims struct {
	StudentID string `json:"sub"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

func jwtSecret() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "dev-secret-change-in-production"
	}
	return []byte(s)
}

// IssueToken creates a signed JWT for the user.
func IssueToken(u *models.User) (string, error) {
	ttl := 24 * time.Hour
	if v := os.Getenv("JWT_TTL_HOURS"); v != "" {
		if h, err := time.ParseDuration(v + "h"); err == nil {
			ttl = h
		}
	}
	claims := Claims{
		StudentID: u.StudentID,
		Role:      u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "edu-lab-platform",
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(jwtSecret())
}

// ParseToken validates Bearer token and returns claims.
func ParseToken(header string) (*Claims, error) {
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return nil, errors.New("missing bearer")
	}
	raw := strings.TrimSpace(header[7:])
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		return jwtSecret(), nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// Login checks student_id + password against DB.
func Login(studentID, password string) (*models.User, error) {
	g := db.DB()
	if g == nil {
		return nil, errors.New("database not configured")
	}
	var u models.User
	if err := g.Where("student_id = ?", studentID).First(&u).Error; err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, errors.New("invalid credentials")
	}
	return &u, nil
}

// Context helpers
func WithClaims(ctx context.Context, c *Claims) context.Context {
	ctx = context.WithValue(ctx, ctxStudentID, c.StudentID)
	ctx = context.WithValue(ctx, ctxRole, c.Role)
	return ctx
}

func StudentIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxStudentID).(string)
	return v
}

func RoleFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxRole).(string)
	return v
}
