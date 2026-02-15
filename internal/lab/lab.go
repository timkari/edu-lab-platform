package lab

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/edu-lab-platform/internal/config"
	"github.com/edu-lab-platform/internal/logger"
)

// ContainerName returns docker container name for student.
func ContainerName(studentID string) string {
	return "lab_" + studentID
}

// FindFreePort finds an available port starting from base port
func FindFreePort(basePort int) (int, error) {
	log := logger.Get()
	for port := basePort; port < basePort+100; port++ {
		// Check if port is in use by any container
		cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("publish=%d", port), "-q")
		out, err := cmd.Output()
		if err != nil {
			continue
		}
		if len(out) == 0 {
			// Also check if port is free on host
			checkPort := exec.Command("sh", "-c", fmt.Sprintf("lsof -i :%d", port))
			if err := checkPort.Run(); err != nil {
				log.Info("Found free port: %d", port)
				return port, nil
			}
		}
	}
	return 0, fmt.Errorf("no free ports found in range %d-%d", basePort, basePort+100)
}

// Start runs Docker container with VNC desktop, mounts student work dir.
func Start(basePath, studentID string) error {
	log := logger.Get()
	log.LogEvent("START_ATTEMPT", studentID, "starting", nil)

	if err := os.MkdirAll(config.WorkDir(basePath, studentID), 0755); err != nil {
		log.Error("Failed to create work dir for %s: %v", studentID, err)
		return err
	}

	workDir, err := filepath.Abs(config.WorkDir(basePath, studentID))
	if err != nil {
		log.Error("Failed to get absolute path for %s: %v", studentID, err)
		return err
	}

	// Find free port for web
	webPort, err := FindFreePort(6080)
	if err != nil {
		log.Error("Failed to find free port for %s: %v", studentID, err)
		return fmt.Errorf("failed to find free web port: %v", err)
	}

	containerName := ContainerName(studentID)

	// Check if container exists and remove if stopped
	checkCmd := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", containerName), "-q")
	existing, _ := checkCmd.Output()
	if len(existing) > 0 {
		log.Warn("Removing existing container %s", containerName)
		removeCmd := exec.Command("docker", "rm", "-f", containerName)
		removeCmd.Run()
	}

	// Run container with dynamic port
	cmd := exec.Command("docker", "run", "-d",
		"--name", containerName,
		"-p", fmt.Sprintf("%d:80", webPort),
		"-v", workDir+":/home/ubuntu/work",
		"-e", fmt.Sprintf("VNC_PW=%s", config.VNCPassword),
		config.LabImage,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("Docker run failed for %s: %v\n%s", studentID, err, string(output))
		return fmt.Errorf("docker run failed: %v\n%s", err, string(output))
	}

	log.Info("Container %s started on port %d", containerName, webPort)
	log.LogEvent("START_SUCCESS", studentID, "running", map[string]string{
		"port":     fmt.Sprintf("%d", webPort),
		"work_dir": workDir,
	})

	return nil
}

// Stop stops and removes the container.
func Stop(studentID string) error {
	log := logger.Get()
	containerName := ContainerName(studentID)

	// Check if container exists
	checkCmd := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", containerName), "-q")
	existing, _ := checkCmd.Output()
	if len(existing) == 0 {
		log.Warn("Container %s does not exist", containerName)
		return nil
	}

	log.Info("Stopping container %s", containerName)
	_ = exec.Command("docker", "stop", containerName).Run()
	_ = exec.Command("docker", "rm", containerName).Run()

	log.LogEvent("STOP_SUCCESS", studentID, "stopped", nil)
	return nil
}

// Info returns URL and password for the lab.
func Info(studentID string) (url, password string) {
	cmd := exec.Command("docker", "port", ContainerName(studentID), "80")
	out, err := cmd.Output()
	if err != nil {
		return "http://localhost:6080", config.VNCPassword
	}

	portStr := strings.TrimSpace(string(out))
	parts := strings.Split(portStr, ":")
	if len(parts) >= 2 {
		return fmt.Sprintf("http://localhost:%s", parts[1]), config.VNCPassword
	}
	return "http://localhost:6080", config.VNCPassword
}

// IsRunning checks if container exists and is running.
func IsRunning(studentID string) (bool, error) {
	out, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", ContainerName(studentID)).Output()
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(string(out)) == "true", nil
}

// WorkDirPath returns absolute path to student work directory.
func WorkDirPath(basePath, studentID string) (string, error) {
	return filepath.Abs(config.WorkDir(basePath, studentID))
}

// GetAllRunning returns list of all running labs
func GetAllRunning() ([]string, error) {
	cmd := exec.Command("docker", "ps", "--filter", "name=lab_", "--format", "{{.Names}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var result []string
	for _, line := range lines {
		if line != "" {
			result = append(result, line)
		}
	}
	return result, nil
}
