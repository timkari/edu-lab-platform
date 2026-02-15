package config

import (
	"path/filepath"
)

const (
	BackupDir   = "backups"
	StudentsDir = "students"
	MaxBackups  = 3
	LabPort     = "8080"
	LabImage    = "dorowu/ubuntu-desktop-lxde-vnc"
	VNCPassword = "vncpassword"
)

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
