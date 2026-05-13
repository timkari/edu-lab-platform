package store

import (
	"github.com/edu-lab-platform/internal/db"
	"github.com/edu-lab-platform/internal/models"
	"gorm.io/gorm"
)

// HasActiveVM — у студента есть сессия не в статусе terminated.
func HasActiveVM(studentID string) (bool, error) {
	g := db.DB()
	if g == nil {
		return false, nil
	}
	var count int64
	err := g.Model(&models.Session{}).
		Where("student_id = ? AND status IN ?", studentID, []string{models.SessionRunning, models.SessionStopped}).
		Count(&count).Error
	return count > 0, err
}

// GetActiveSession returns first non-terminated session for student, if any.
func GetActiveSession(studentID string) (*models.Session, error) {
	g := db.DB()
	if g == nil {
		return nil, nil
	}
	var s models.Session
	err := g.Where("student_id = ? AND status IN ?", studentID, []string{models.SessionRunning, models.SessionStopped}).
		Order("id DESC").
		First(&s).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// PendingCreateRequestExists — есть необработанная заявка на создание.
func PendingCreateRequestExists(studentID string) (bool, error) {
	g := db.DB()
	if g == nil {
		return false, nil
	}
	var count int64
	err := g.Model(&models.Request{}).
		Where("student_id = ? AND type = ? AND status = ?", studentID, models.RequestTypeCreate, models.RequestStatusPending).
		Count(&count).Error
	return count > 0, err
}
