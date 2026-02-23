package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	urlpkg "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

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
	StudentID  string `json:"student_id"`
	BackupFile string `json:"backup_file"`
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

	// Get the actual URL with dynamic port (from Docker)
	labURL, password := lab.Info(req.StudentID)

	// Перестраиваем URL, чтобы использовать тот же хост, по которому пользователь
	// открывает веб-интерфейс (r.Host), а не localhost внутри контейнера.
	if r.Host != "" {
		if u, err := urlpkg.Parse(labURL); err == nil {
			host, _, errHost := net.SplitHostPort(r.Host)
			if errHost != nil {
				// r.Host без порта — используем как есть
				host = r.Host
			}

			vncPort := u.Port()
			if vncPort != "" && host != "" {
				u.Host = net.JoinHostPort(host, vncPort)
				labURL = u.String()
			}
		}
	}

	workDir, _ := lab.WorkDirPath(base, req.StudentID)

	writeJSON(w, http.StatusOK, Response{OK: true, Data: map[string]string{
		"url":      labURL,
		"password": password,
		"work_dir": workDir,
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

// НОВАЯ ФУНКЦИЯ: HandleStatus проверяет, запущена ли лаборатория
func HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{OK: false, Error: "GET only"})
		return
	}

	studentID := r.URL.Query().Get("student_id")
	if studentID == "" {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: "student_id обязателен"})
		return
	}

	running, err := lab.IsRunning(studentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{OK: true, Data: map[string]interface{}{
		"running":    running,
		"student_id": studentID,
	}})
}

// НОВАЯ ФУНКЦИЯ: HandleList возвращает список всех студентов и их статусы
func HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{OK: false, Error: "GET only"})
		return
	}

	base := BasePath()
	studentsDir := filepath.Join(base, "students")

	// Читаем папку students
	entries, err := os.ReadDir(studentsDir)
	if err != nil {
		writeJSON(w, http.StatusOK, Response{OK: true, Data: []interface{}{}})
		return
	}

	var students []map[string]interface{}
	for _, entry := range entries {
		if entry.IsDir() {
			studentID := entry.Name()
			running, _ := lab.IsRunning(studentID)

			// Получаем список бэкапов для студента
			backupPattern := filepath.Join(base, "backups", studentID+"_*.tar.gz")
			backups, _ := filepath.Glob(backupPattern)
			backupFiles := make([]string, 0)
			for _, b := range backups {
				backupFiles = append(backupFiles, filepath.Base(b))
			}

			students = append(students, map[string]interface{}{
				"id":       studentID,
				"running":  running,
				"backups":  backupFiles,
				"work_dir": filepath.Join(studentsDir, studentID, "work"),
			})
		}
	}

	writeJSON(w, http.StatusOK, Response{OK: true, Data: students})
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

// HandleGetLogs returns recent logs
func HandleGetLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{OK: false, Error: "GET only"})
		return
	}

	// Get log file path
	logDir := filepath.Join(BasePath(), "logs")
	logFile := filepath.Join(logDir, fmt.Sprintf("lab_%s.log", time.Now().Format("2006-01-02")))

	// Read last 100 lines
	cmd := exec.Command("tail", "-n", "100", logFile)
	out, err := cmd.Output()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{OK: true, Data: string(out)})
}

// Mux returns http.ServeMux with all routes and static frontend.
func Mux() *http.ServeMux {
	m := http.NewServeMux()
	m.HandleFunc("/api/start", HandleStart)
	m.HandleFunc("/api/stop", HandleStop)
	m.HandleFunc("/api/backup", HandleBackup)
	m.HandleFunc("/api/restore", HandleRestore)
	m.HandleFunc("/api/structure", HandleStructure)
	m.HandleFunc("/api/logs", HandleGetLogs)
	m.HandleFunc("/api/status", HandleStatus) // НОВЫЙ МАРШРУТ
	m.HandleFunc("/api/list", HandleList)     // НОВЫЙ МАРШРУТ для админки
	m.Handle("/", http.FileServer(http.Dir(WebDir())))

	return m
}
