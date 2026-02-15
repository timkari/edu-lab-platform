package lab

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
		// Способ 1: Проверка через Docker (быстро, но не всегда точно)
		cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("publish=%d", port), "-q")
		out, err := cmd.Output()
		if err != nil {
			log.Debug("Docker port check failed for %d: %v", port, err)
			continue
		}
		
		// Если порт занят каким-то контейнером
		if len(out) > 0 {
			log.Debug("Port %d is used by Docker container", port)
			continue
		}
		
		// Способ 2: Проверка через net.Listen (более надежно)
		if isPortAvailable(port) {
			log.Info("Found free port: %d", port)
			return port, nil
		}
		
		log.Debug("Port %d is not available", port)
	}
	
	return 0, fmt.Errorf("no free ports found in range %d-%d", basePort, basePort+100)
}

// isPortAvailable проверяет, доступен ли порт для прослушивания
func isPortAvailable(port int) bool {
	// Пытаемся открыть TCP соединение на порту
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	ln.Close()
	
	// Даем небольшой таймаут для освобождения порта
	time.Sleep(50 * time.Millisecond)
	
	return true
}

// Start runs Docker container with VNC desktop, mounts student work dir.
func Start(basePath, studentID string) error {
	log := logger.Get()
	log.LogEvent("START_ATTEMPT", studentID, "starting", nil)

	// Создаем рабочую директорию
	if err := os.MkdirAll(config.WorkDir(basePath, studentID), 0755); err != nil {
		log.Error("Failed to create work dir for %s: %v", studentID, err)
		return err
	}

	workDir, err := filepath.Abs(config.WorkDir(basePath, studentID))
	if err != nil {
		log.Error("Failed to get absolute path for %s: %v", studentID, err)
		return err
	}

	// Находим свободный порт
	webPort, err := FindFreePort(6080)
	if err != nil {
		log.Error("Failed to find free port for %s: %v", studentID, err)
		// Пробуем альтернативный диапазон портов
		webPort, err = FindFreePort(8080)
		if err != nil {
			return fmt.Errorf("failed to find free web port: %v", err)
		}
	}

	containerName := ContainerName(studentID)

	// Проверяем существующий контейнер
	existing, _ := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", containerName), "-q").Output()
	if len(existing) > 0 {
		log.Warn("Removing existing container %s", containerName)
		exec.Command("docker", "rm", "-f", containerName).Run()
	}

	// Запускаем контейнер
	cmd := exec.Command("docker", "run", "-d",
		"--name", containerName,
		"--restart", "no",
		"-p", fmt.Sprintf("%d:80", webPort),
		"-v", workDir+":/home/ubuntu/work",
		"-e", fmt.Sprintf("VNC_PW=%s", config.VNCPassword),
		"--memory", "512m",
		"--memory-swap", "512m",
		"--cpus", "0.5",
		config.LabImage,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("Docker run failed for %s: %v\n%s", studentID, err, string(output))
		return fmt.Errorf("docker run failed: %v\n%s", err, string(output))
	}

	// Ждем, пока контейнер полностью запустится
	time.Sleep(2 * time.Second)

	// Проверяем, что контейнер действительно запущен
	running, err := IsRunning(studentID)
	if err != nil || !running {
		log.Error("Container %s is not running after start", containerName)
		return fmt.Errorf("container failed to start")
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

	// Проверяем существование контейнера
	out, _ := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", containerName), "-q").Output()
	if len(out) == 0 {
		log.Warn("Container %s does not exist", containerName)
		return nil
	}

	log.Info("Stopping container %s", containerName)
	
	// Останавливаем контейнер с таймаутом
	stopCmd := exec.Command("docker", "stop", "--time", "10", containerName)
	if err := stopCmd.Run(); err != nil {
		log.Warn("Failed to stop container gracefully: %v", err)
		// Принудительно удаляем
		exec.Command("docker", "rm", "-f", containerName).Run()
	} else {
		// Удаляем контейнер после остановки
		exec.Command("docker", "rm", containerName).Run()
	}

	log.LogEvent("STOP_SUCCESS", studentID, "stopped", nil)
	return nil
}

// Info returns URL and password for the lab.
func Info(studentID string) (url, password string) {
	containerName := ContainerName(studentID)
	
	// Получаем информацию о порте
	cmd := exec.Command("docker", "port", containerName, "80")
	out, err := cmd.Output()
	if err != nil {
		// Если не удалось получить порт, возвращаем значения по умолчанию
		return "http://localhost:6080", config.VNCPassword
	}

	portStr := strings.TrimSpace(string(out))
	if portStr == "" {
		return "http://localhost:6080", config.VNCPassword
	}

	// Парсим вывод "0.0.0.0:6081" или подобное
	parts := strings.Split(portStr, ":")
	if len(parts) >= 2 {
		hostPort := parts[len(parts)-1]
		// Проверяем, что порт - число
		if _, err := strconv.Atoi(hostPort); err == nil {
			return fmt.Sprintf("http://localhost:%s", hostPort), config.VNCPassword
		}
	}

	return "http://localhost:6080", config.VNCPassword
}

// IsRunning checks if container exists and is running.
func IsRunning(studentID string) (bool, error) {
	containerName := ContainerName(studentID)
	
	// Проверяем статус контейнера
	out, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", containerName).Output()
	if err != nil {
		// Контейнер не существует или другая ошибка
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
		if line != "" && strings.HasPrefix(line, "lab_") {
			result = append(result, line)
		}
	}
	return result, nil
}

// GetContainerPort возвращает реальный порт контейнера
func GetContainerPort(studentID string) (int, error) {
	containerName := ContainerName(studentID)
	
	// Получаем информацию о порте
	cmd := exec.Command("docker", "port", containerName, "80")
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get port: %v", err)
	}

	portStr := strings.TrimSpace(string(out))
	if portStr == "" {
		return 0, fmt.Errorf("no port mapping found")
	}

	// Парсим вывод
	parts := strings.Split(portStr, ":")
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid port format: %s", portStr)
	}

	hostPort := parts[len(parts)-1]
	port, err := strconv.Atoi(hostPort)
	if err != nil {
		return 0, fmt.Errorf("invalid port number: %s", hostPort)
	}

	return port, nil
}