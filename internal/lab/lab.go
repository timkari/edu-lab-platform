package lab

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/edu-lab-platform/internal/config"
)

// ContainerName returns docker container name for student.
func ContainerName(studentID string) string {
	return "lab_" + studentID
}

// Start runs Docker container with VNC desktop, mounts student work dir.
func Start(basePath, studentID string) error {
	if err := os.MkdirAll(config.WorkDir(basePath, studentID), 0755); err != nil {
		return err
	}
	workDir, err := filepath.Abs(config.WorkDir(basePath, studentID))
	if err != nil {
		return err
	}
	cmd := exec.Command("docker", "run", "-d",
		"--name", ContainerName(studentID),
		"-p", config.LabPort+":80",
		"-v", workDir+":/home/ubuntu/work",
		config.LabImage,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Stop stops and removes the container.
func Stop(studentID string) error {
	_ = exec.Command("docker", "stop", ContainerName(studentID)).Run()
	_ = exec.Command("docker", "rm", ContainerName(studentID)).Run()
	return nil
}

// Info returns URL and password for the lab.
func Info(studentID string) (url, password string) {
	return "http://localhost:" + config.LabPort, config.VNCPassword
}

// IsRunning checks if container exists and is running.
func IsRunning(studentID string) (bool, error) {
	out, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", ContainerName(studentID)).Output()
	if err != nil {
		return false, nil
	}
	return string(out) == "true\n", nil
}

// WorkDirPath returns absolute path to student work directory.
func WorkDirPath(basePath, studentID string) (string, error) {
	return filepath.Abs(config.WorkDir(basePath, studentID))
}
