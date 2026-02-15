package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Log levels
const (
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"
)

// Logger struct
type Logger struct {
	infoLog  *log.Logger
	warnLog  *log.Logger
	errorLog *log.Logger
	logFile  *os.File
}

var instance *Logger

// Init initializes the logger
func Init(logDir string) error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// Create log file with current date
	filename := filepath.Join(logDir, fmt.Sprintf("lab_%s.log", time.Now().Format("2006-01-02")))
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	instance = &Logger{
		infoLog:  log.New(file, "[INFO] ", log.Ldate|log.Ltime),
		warnLog:  log.New(file, "[WARN] ", log.Ldate|log.Ltime),
		errorLog: log.New(file, "[ERROR] ", log.Ldate|log.Ltime),
		logFile:  file,
	}

	// Also log to console
	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime)

	return nil
}

// Get returns the logger instance
func Get() *Logger {
	if instance == nil {
		// Fallback to default logger
		return &Logger{
			infoLog:  log.New(os.Stdout, "[INFO] ", log.Ldate|log.Ltime),
			warnLog:  log.New(os.Stdout, "[WARN] ", log.Ldate|log.Ltime),
			errorLog: log.New(os.Stderr, "[ERROR] ", log.Ldate|log.Ltime),
		}
	}
	return instance
}

// Info logs info message
func (l *Logger) Info(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.infoLog.Println(msg)
}

// Warn logs warning message
func (l *Logger) Warn(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.warnLog.Println(msg)
}

// Error logs error message
func (l *Logger) Error(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.errorLog.Println(msg)
}

// Debug logs debug message
func (l *Logger) Debug(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.infoLog.Println("[DEBUG] " + msg)
}

// Close closes the log file
func (l *Logger) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
}

// LogEvent logs a structured event
func (l *Logger) LogEvent(eventType, studentID, status string, details map[string]string) {
	event := fmt.Sprintf("EVENT: %s | student: %s | status: %s", eventType, studentID, status)
	for k, v := range details {
		event += fmt.Sprintf(" | %s: %s", k, v)
	}
	l.infoLog.Println(event)
}