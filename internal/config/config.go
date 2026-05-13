package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	BackupDir   = "backups"
	StudentsDir = "students"
	MaxBackups  = 3
	LabPort     = "8080"
	// DefaultLabImage — образ по умолчанию, если не задана переменная LAB_IMAGE.
	DefaultLabImage = "dorowu/ubuntu-desktop-lxde-vnc"
	VNCPassword     = "vncpassword"
)

// LabDockerImage возвращает имя Docker-образа лаборатории (шаблон по умолчанию и lab.Start при пустом образе).
func LabDockerImage() string {
	if v := strings.TrimSpace(os.Getenv("LAB_IMAGE")); v != "" {
		return v
	}
	return DefaultLabImage
}

// SessionIdleTimeout — без успешного ping дольше этого интервала ВМ останавливается (фоновая задача).
// Переменная SESSION_IDLE_MINUTES, по умолчанию 60.
func SessionIdleTimeout() time.Duration {
	if v := strings.TrimSpace(os.Getenv("SESSION_IDLE_MINUTES")); v != "" {
		if m, err := strconv.Atoi(v); err == nil && m > 0 {
			return time.Duration(m) * time.Minute
		}
	}
	return time.Hour
}

// WorkDir returns students/{id}/work
func WorkDir(basePath, studentID string) string {
	return filepath.Join(basePath, StudentsDir, studentID, "work")
}

// StudentPath returns students/{id}
func StudentPath(basePath, studentID string) string {
	return filepath.Join(basePath, StudentsDir, studentID)
}

// BackupPath returns backups/{id}_{timestamp}.tar.gz
func BackupPath(basePath, studentID, timestamp string) string {
	return filepath.Join(basePath, BackupDir, studentID+"_"+timestamp+".tar.gz")
}
