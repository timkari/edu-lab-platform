package idle

import (
	"sync"
	"time"

	"github.com/edu-lab-platform/internal/backup"
	"github.com/edu-lab-platform/internal/config"
	"github.com/edu-lab-platform/internal/db"
	"github.com/edu-lab-platform/internal/lab"
	"github.com/edu-lab-platform/internal/logger"
	"github.com/edu-lab-platform/internal/models"
	"gorm.io/gorm"
)

// StartReaper периодически завершает сессии без «пульса» от студента дольше SessionIdleTimeout().
func StartReaper(basePath string) {
	go func() {
		tick(basePath)
		t := time.NewTicker(2 * time.Minute)
		defer t.Stop()
		for range t.C {
			tick(basePath)
		}
	}()
}

var tickMu sync.Mutex

func tick(basePath string) {
	if !tickMu.TryLock() {
		return
	}
	defer tickMu.Unlock()

	g := db.DB()
	if g == nil {
		return
	}
	log := logger.Get()
	timeout := config.SessionIdleTimeout()

	var ids []uint
	if err := g.Model(&models.Session{}).
		Where("status IN ?", []string{models.SessionRunning, models.SessionStopped}).
		Pluck("id", &ids).Error; err != nil {
		log.Error("idle reaper: list sessions: %v", err)
		return
	}
	for _, id := range ids {
		tryTerminateStale(g, basePath, id, timeout, log)
	}
}

func tryTerminateStale(g *gorm.DB, basePath string, id uint, timeout time.Duration, lg *logger.Logger) {
	var cur models.Session
	if err := g.First(&cur, id).Error; err != nil {
		return
	}
	if cur.Status != models.SessionRunning && cur.Status != models.SessionStopped {
		return
	}
	ref := cur.LastSeenAt
	if ref.IsZero() {
		ref = cur.CreatedAt
	}
	if time.Since(ref) < timeout {
		return
	}

	lg.Info("idle reaper: сессия %d (%s) без активности > %v — остановка ВМ", id, cur.StudentID, timeout)
	_, _ = backup.Create(basePath, cur.StudentID)
	_ = lab.Stop(cur.StudentID)
	cur.Status = models.SessionTerminated
	if err := g.Save(&cur).Error; err != nil {
		lg.Error("idle reaper: save session %d: %v", id, err)
	}
}
