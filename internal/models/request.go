package models

import "time"

const (
	RequestTypeCreate = "create"
	RequestTypeDelete = "delete"

	RequestStatusPending   = "pending"
	RequestStatusApproved  = "approved"
	RequestStatusRejected  = "rejected"
	RequestStatusCancelled = "cancelled"
)

// Request — заявка на создание или удаление ВМ.
type Request struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	StudentID    string    `gorm:"type:varchar(50);not null;index" json:"student_id"`
	TemplateID   uint      `gorm:"not null" json:"template_id"`
	Description  string    `gorm:"type:text;not null" json:"description"`
	Type         string    `gorm:"type:varchar(20);not null;default:create" json:"type"`
	Status       string    `gorm:"type:varchar(20);not null;default:pending" json:"status"`
	AdminComment string    `gorm:"type:text" json:"admin_comment,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
