package models

import "time"

// Статусы сессии ВМ (см. ТЗ: активна, если не terminated).
const (
	SessionRunning    = "running"
	SessionStopped    = "stopped"
	SessionTerminated = "terminated"
)

// Session — привязка студента к запущенной лаборатории и заявке.
type Session struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	StudentID  string    `gorm:"type:varchar(50);not null;index" json:"student_id"`
	TemplateID uint      `gorm:"not null" json:"template_id"`
	RequestID  *uint     `gorm:"index" json:"request_id,omitempty"`
	Status     string    `gorm:"type:varchar(20);not null;default:running" json:"status"`
	AccessURL  string    `gorm:"type:varchar(500)" json:"access_url"`
	// LastSeenAt обновляется POST /api/session/ping (пока студент «в сети» в кабинете).
	LastSeenAt time.Time `gorm:"index" json:"last_seen_at,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
