package db

import (
	"fmt"
	"os"
	"strings"

	"github.com/edu-lab-platform/internal/config"
	"github.com/edu-lab-platform/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var global *gorm.DB

// DB returns the global GORM handle (nil if not initialized).
func DB() *gorm.DB {
	return global
}

// Init connects to PostgreSQL and runs AutoMigrate + seed.
func Init() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}
	g, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return err
	}
	if err := g.AutoMigrate(
		&models.User{},
		&models.Template{},
		&models.Session{},
		&models.Request{},
	); err != nil {
		return err
	}
	global = g
	return seed()
}

func seed() error {
	if global == nil {
		return nil
	}
	var tc int64
	if err := global.Model(&models.Template{}).Count(&tc).Error; err != nil {
		return err
	}
	if tc == 0 {
		if err := global.Create(&models.Template{
			Name:        "Ubuntu Desktop (VNC)",
			Description: "Стандартная лаборатория LXDE + noVNC, Geany на рабочем столе",
			DockerImage: config.LabDockerImage(),
		}).Error; err != nil {
			return err
		}
	}
	if err := syncTemplateLabImageFromEnv(); err != nil {
		return err
	}
	var uc int64
	if err := global.Model(&models.User{}).Count(&uc).Error; err != nil {
		return err
	}
	if uc > 0 {
		return nil
	}
	adminHash, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	studentHash, err := bcrypt.GenerateFromPassword([]byte("student"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	users := []models.User{
		{StudentID: "admin", PasswordHash: string(adminHash), Role: "admin"},
		{StudentID: "student1", PasswordHash: string(studentHash), Role: "student"},
	}
	return global.Create(&users).Error
}

// syncTemplateLabImageFromEnv обновляет образ у шаблона по умолчанию, если задан LAB_IMAGE (например после первой сборки custom-образа).
func syncTemplateLabImageFromEnv() error {
	if global == nil {
		return nil
	}
	v := strings.TrimSpace(os.Getenv("LAB_IMAGE"))
	if v == "" {
		return nil
	}
	return global.Model(&models.Template{}).
		Where("name = ?", "Ubuntu Desktop (VNC)").
		Update("docker_image", v).Error
}
