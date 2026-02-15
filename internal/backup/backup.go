package backup

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/edu-lab-platform/internal/config"
)

// Create creates a tar.gz backup of student data. basePath is project root.
func Create(basePath, studentID string) (string, error) {
	studentPath := config.StudentPath(basePath, studentID)
	if _, err := os.Stat(studentPath); os.IsNotExist(err) {
		return "", fmt.Errorf("нет данных: %s", studentPath)
	}

	timestamp := time.Now().Format("20060102_150405")
	backupPath := config.BackupPath(basePath, studentID, timestamp)

	backupDir := filepath.Join(basePath, config.BackupDir)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}

	f, err := os.Create(backupPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	base := filepath.Join(basePath, config.StudentsDir)
	err = filepath.Walk(studentPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if info.IsDir() {
			rel += "/"
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = rel
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if !info.IsDir() {
			fd, err := os.Open(path)
			if err != nil {
				return err
			}
			defer fd.Close()
			_, err = io.Copy(tw, fd)
			return err
		}
		return nil
	})
	if err != nil {
		os.Remove(backupPath)
		return "", err
	}

	CleanupOld(basePath, studentID)
	return backupPath, nil
}

// CleanupOld removes old backups for student, keeping MaxBackups.
func CleanupOld(basePath, studentID string) {
	pattern := filepath.Join(basePath, config.BackupDir, studentID+"_*.tar.gz")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}
	if len(matches) <= config.MaxBackups {
		return
	}
	sort.Slice(matches, func(i, j int) bool {
		// by mtime descending (newest first)
		ii, _ := os.Stat(matches[i])
		jj, _ := os.Stat(matches[j])
		return ii.ModTime().After(jj.ModTime())
	})
	for _, p := range matches[config.MaxBackups:] {
		os.Remove(p)
	}
}

// Restore extracts backup into students/{studentID}. Removes existing student dir first.
func Restore(basePath, studentID, backupPath string) error {
	f, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("файл не найден: %s", backupPath)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	studentPath := config.StudentPath(basePath, studentID)
	os.RemoveAll(studentPath)
	if err := os.MkdirAll(filepath.Dir(studentPath), 0755); err != nil {
		return err
	}

	tr := tar.NewReader(gr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// name in archive is like "studentID/work/..."
		target := filepath.Join(basePath, config.StudentsDir, h.Name)
		if h.Typeflag == tar.TypeDir {
			os.MkdirAll(target, 0755)
			continue
		}
		os.MkdirAll(filepath.Dir(target), 0755)
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, tr)
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// EnsureStructure creates backups and students directories.
func EnsureStructure(basePath string) error {
	dirs := []string{
		filepath.Join(basePath, config.BackupDir),
		filepath.Join(basePath, config.StudentsDir),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}
