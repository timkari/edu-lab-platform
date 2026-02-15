package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/edu-lab-platform/internal/backup"
	"github.com/edu-lab-platform/internal/lab"
)

// BasePath returns project root (where backups/ and students/ live).
func BasePath() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}

// StartLabRequest body.
type StartLabRequest struct {
	StudentID string `json:"student_id"`
}

// StopLabRequest body.
type StopLabRequest struct {
	StudentID string `json:"student_id"`
}

// RestoreRequest body.
type RestoreRequest struct {
	StudentID   string `json:"student_id"`
	BackupFile  string `json:"backup_file"`
}

// Response common.
type Response struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	Data  any    `json:"data,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// HandleStart starts lab for student.
func HandleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{OK: false, Error: "POST only"})
		return
	}
	var req StartLabRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	if req.StudentID == "" {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: "student_id обязателен"})
		return
	}
	base := BasePath()
	if err := lab.Start(base, req.StudentID); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	url, password := lab.Info(req.StudentID)
	workDir, _ := lab.WorkDirPath(base, req.StudentID)
	writeJSON(w, http.StatusOK, Response{OK: true, Data: map[string]string{
		"url":       url,
		"password":  password,
		"work_dir":  workDir,
	}})
}

// HandleStop stops lab and creates backup.
func HandleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{OK: false, Error: "POST only"})
		return
	}
	var req StopLabRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	if req.StudentID == "" {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: "student_id обязателен"})
		return
	}
	base := BasePath()
	if path, err := backup.Create(base, req.StudentID); err == nil {
		_ = path
	}
	lab.Stop(req.StudentID)
	writeJSON(w, http.StatusOK, Response{OK: true})
}

// HandleBackup creates backup for student.
func HandleBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{OK: false, Error: "POST only"})
		return
	}
	var req struct {
		StudentID string `json:"student_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	if req.StudentID == "" {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: "student_id обязателен"})
		return
	}
	path, err := backup.Create(BasePath(), req.StudentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: map[string]string{"backup_file": path}})
}

// HandleRestore restores from backup.
func HandleRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{OK: false, Error: "POST only"})
		return
	}
	var req RestoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	if req.StudentID == "" || req.BackupFile == "" {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: "student_id и backup_file обязательны"})
		return
	}
	path := req.BackupFile
	if !filepath.IsAbs(path) {
		path = filepath.Join(BasePath(), path)
	}
	if err := backup.Restore(BasePath(), req.StudentID, path); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true})
}

// HandleStructure creates directories.
func HandleStructure(w http.ResponseWriter, r *http.Request) {
	if err := backup.EnsureStructure(BasePath()); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true})
}

// WebDir returns path to web static files (next to executable, in cwd, or in cwd/edu-lab-platform).
func WebDir() string {
	base := BasePath()
	try := []string{
		filepath.Join(base, "web"),
		filepath.Join(base, "edu-lab-platform", "web"),
	}
	if execPath, err := os.Executable(); err == nil {
		try = append([]string{filepath.Join(filepath.Dir(execPath), "web")}, try...)
	}
	for _, d := range try {
		if _, err := os.Stat(filepath.Join(d, "index.html")); err == nil {
			return d
		}
	}
	return filepath.Join(base, "web")
}

// Mux returns http.ServeMux with all routes and static frontend.
func Mux() *http.ServeMux {
	m := http.NewServeMux()
	m.HandleFunc("/api/start", HandleStart)
	m.HandleFunc("/api/stop", HandleStop)
	m.HandleFunc("/api/backup", HandleBackup)
	m.HandleFunc("/api/restore", HandleRestore)
	m.HandleFunc("/api/structure", HandleStructure)
	m.Handle("/", http.FileServer(http.Dir(WebDir())))
	return m
}
