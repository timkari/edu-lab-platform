package tests

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/edu-lab-platform/internal/config"
)

func TestSessionIdleTimeout_Default(t *testing.T) {
	t.Setenv("SESSION_IDLE_MINUTES", "")
	d := config.SessionIdleTimeout()
	if d != time.Hour {
		t.Fatalf("default: want %v, got %v", time.Hour, d)
	}
}

func TestSessionIdleTimeout_CustomMinutes(t *testing.T) {
	t.Setenv("SESSION_IDLE_MINUTES", "30")
	d := config.SessionIdleTimeout()
	if d != 30*time.Minute {
		t.Fatalf("custom 30m: got %v", d)
	}
}

func TestSessionIdleTimeout_InvalidFallsBackToDefault(t *testing.T) {
	t.Setenv("SESSION_IDLE_MINUTES", "not-a-number")
	d := config.SessionIdleTimeout()
	if d != time.Hour {
		t.Fatalf("invalid env: want default %v, got %v", time.Hour, d)
	}
}

func TestSessionIdleTimeout_ZeroOrNegativeFallsBack(t *testing.T) {
	t.Setenv("SESSION_IDLE_MINUTES", "0")
	if config.SessionIdleTimeout() != time.Hour {
		t.Fatal("0 minutes should fall back to default")
	}
	t.Setenv("SESSION_IDLE_MINUTES", "-5")
	if config.SessionIdleTimeout() != time.Hour {
		t.Fatal("negative minutes should fall back to default")
	}
}

func TestLabDockerImage_Default(t *testing.T) {
	t.Setenv("LAB_IMAGE", "")
	if got := config.LabDockerImage(); got != config.DefaultLabImage {
		t.Fatalf("want %q, got %q", config.DefaultLabImage, got)
	}
}

func TestLabDockerImage_FromEnv(t *testing.T) {
	t.Setenv("LAB_IMAGE", "  my-registry/lab:v2  ")
	if got := config.LabDockerImage(); got != "my-registry/lab:v2" {
		t.Fatalf("want trimmed image, got %q", got)
	}
}

func TestWorkDir(t *testing.T) {
	base := "/app"
	got := config.WorkDir(base, "student1")
	want := filepath.Join(base, "students", "student1", "work")
	if got != want {
		t.Fatalf("WorkDir: want %q, got %q", want, got)
	}
}

func TestStudentPath(t *testing.T) {
	base := "/data"
	got := config.StudentPath(base, "u42")
	want := filepath.Join(base, "students", "u42")
	if got != want {
		t.Fatalf("StudentPath: want %q, got %q", want, got)
	}
}

func TestBackupPath(t *testing.T) {
	base := "/x"
	got := config.BackupPath(base, "s1", "20260101120000")
	want := filepath.Join(base, "backups", "s1_20260101120000.tar.gz")
	if got != want {
		t.Fatalf("BackupPath: want %q, got %q", want, got)
	}
}
