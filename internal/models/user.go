package models

import "time"

// User — учётная запись по номеру студенческого билета (student_id) или админ.
type User struct {
	StudentID    string    `gorm:"primaryKey;type:varchar(50)" json:"student_id"`
	PasswordHash string    `gorm:"type:varchar(255);not null" json:"-"`
	Role         string    `gorm:"type:varchar(20);not null;default:student" json:"role"` // student | admin
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
