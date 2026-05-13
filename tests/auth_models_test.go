package tests

import (
	"context"
	"strings"
	"testing"

	"github.com/edu-lab-platform/internal/auth"
	"github.com/edu-lab-platform/internal/models"
)

func TestParseToken_MissingBearer(t *testing.T) {
	for _, h := range []string{"", "Token xyz", "Basic x"} {
		_, err := auth.ParseToken(h)
		if err == nil || !strings.Contains(err.Error(), "missing bearer") {
			t.Fatalf("header %q: want missing bearer error, got %v", h, err)
		}
	}
}

func TestIssueToken_ParseToken_RoundTrip(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-for-unit-tests-only")
	t.Setenv("JWT_TTL_HOURS", "")

	u := &models.User{StudentID: "student9", Role: "student", PasswordHash: "-"}
	raw, err := auth.IssueToken(u)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := auth.ParseToken("Bearer " + raw)
	if err != nil {
		t.Fatal(err)
	}
	if claims.StudentID != "student9" || claims.Role != "student" {
		t.Fatalf("claims: %+v", claims)
	}
}

func TestContextClaimsHelpers(t *testing.T) {
	ctx := context.Background()
	c := &auth.Claims{StudentID: "s", Role: "student"}
	ctx = auth.WithClaims(ctx, c)
	if auth.StudentIDFromContext(ctx) != "s" {
		t.Fatal("StudentIDFromContext")
	}
	if auth.RoleFromContext(ctx) != "student" {
		t.Fatal("RoleFromContext")
	}
	if auth.StudentIDFromContext(context.Background()) != "" {
		t.Fatal("empty context should return empty student id")
	}
}

func TestModelsSessionStatuses(t *testing.T) {
	if models.SessionRunning != "running" {
		t.Fatal(models.SessionRunning)
	}
	if models.SessionStopped != "stopped" {
		t.Fatal(models.SessionStopped)
	}
	if models.SessionTerminated != "terminated" {
		t.Fatal(models.SessionTerminated)
	}
}

func TestModelsRequestTypes(t *testing.T) {
	if models.RequestTypeCreate != "create" {
		t.Fatal(models.RequestTypeCreate)
	}
	if models.RequestTypeDelete != "delete" {
		t.Fatal(models.RequestTypeDelete)
	}
}
