package models

import "time"

// Template — шаблон ВМ (Docker-образ).
type Template struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	Name        string    `gorm:"type:varchar(200);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	DockerImage string    `gorm:"type:varchar(500);not null" json:"docker_image"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
