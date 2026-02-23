package lab

import (
	"fmt"
	"net"
	"os/exec"
	"os"
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
    
    log.Info("Searching for free port starting from %d", basePort)
    
    for port := basePort; port < basePort+1000; port++ {
        // Проверяем, не занят ли порт другими контейнерами
        checkCmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("publish=%d", port), "-q")
        checkOut, _ := checkCmd.Output()
        if len(checkOut) > 0 {
            log.Debug("Port %d is used by another container", port)
            continue
        }
        
        // Проверяем, доступен ли порт для прослушивания
        if isPortAvailable(port) {
            log.Info("Found free port: %d", port)
            return port, nil
        }
        
        log.Debug("Port %d is not available", port)
    }
    
    return 0, fmt.Errorf("no free ports found in range %d-%d", basePort, basePort+1000)
}

// isPortAvailable проверяет, доступен ли порт для прослушивания
func isPortAvailable(port int) bool {
    addr := fmt.Sprintf(":%d", port)
    
    // Пробуем открыть порт для прослушивания
    ln, err := net.Listen("tcp", addr)
    if err != nil {
        return false
    }
    ln.Close()
    
    // Даем время на освобождение
    time.Sleep(50 * time.Millisecond)
    
    // Дополнительно проверяем через Docker
    checkCmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("publish=%d", port), "-q")
    out, _ := checkCmd.Output()
    
    return len(out) == 0
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

	// Находим свободный порт - начинаем с 8081 как в тесте
	webPort, err := FindFreePort(8081) // Используем 8081 как в тесте
	if err != nil {
		log.Error("Failed to find free port for %s: %v", studentID, err)
		// Пробуем другой диапазон
		webPort, err = FindFreePort(10000)
		if err != nil {
			return fmt.Errorf("failed to find free web port: %v", err)
		}
	}

	log.Info("Selected port %d for student %s", webPort, studentID)

	// Проверим, действительно ли порт свободен
	if !isPortAvailable(webPort) {
		log.Warn("Port %d is not available according to isPortAvailable, but FindFreePort returned it", webPort)
	}

	// Проверим через Docker
	dockerCheckCmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("publish=%d", webPort), "-q")
	dockerOut, _ := dockerCheckCmd.Output()
	if len(dockerOut) > 0 {
		log.Warn("Port %d is used by Docker containers according to docker ps", webPort)
	}

	containerName := ContainerName(studentID)

	// Проверяем существующий контейнер
	existing, _ := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", containerName), "-q").Output()
	if len(existing) > 0 {
		log.Warn("Removing existing container %s", containerName)
		exec.Command("docker", "rm", "-f", containerName).Run()
	}

	// Запускаем контейнер
	runCmd := exec.Command("docker", "run", "-d",
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

	output, err := runCmd.CombinedOutput()
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
// Info возвращает URL для доступа ИЗ ХОСТА
func Info(studentID string) (url, password string) {
    containerName := ContainerName(studentID)

    cmd := exec.Command("docker", "port", containerName, "80")
    out, err := cmd.Output()
    if err != nil {
        return "http://localhost:6080", config.VNCPassword
    }

    portStr := strings.TrimSpace(string(out))
    if portStr == "" {
        return "http://localhost:6080", config.VNCPassword
    }

    parts := strings.Split(portStr, ":")
    if len(parts) >= 2 {
        hostPort := parts[len(parts)-1]
        if _, err := strconv.Atoi(hostPort); err == nil {
            // ✅ Берём IP из переменной окружения SERVER_IP
            serverIP := os.Getenv("SERVER_IP")
            if serverIP == "" {
                serverIP = "localhost" // fallback для разработки
            }
            return fmt.Sprintf("http://%s:%s", serverIP, hostPort), config.VNCPassword
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