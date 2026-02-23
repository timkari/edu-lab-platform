package lab

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// Platform type
type Platform string

const (
	PlatformMac     Platform = "mac"
	PlatformLinux   Platform = "linux"
	PlatformWindows Platform = "windows"
	PlatformAuto    Platform = "auto"
)

// DetectPlatform определяет текущую платформу
func DetectPlatform() Platform {
	// Проверяем переменную окружения
	platform := os.Getenv("PLATFORM")
	if platform != "" && platform != "auto" {
		return Platform(platform)
	}

	// Автоопределение
	if runtime.GOOS == "darwin" {
		return PlatformMac
	}
	if runtime.GOOS == "linux" {
		// Проверяем, не WSL ли это
		if _, err := os.Stat("/proc/sys/fs/binfmt_misc/WSLInterop"); err == nil {
			return PlatformWindows // WSL считается Windows для совместимости
		}
		return PlatformLinux
	}
	return PlatformWindows
}

// UseDockerVolumes определяет, нужно ли использовать Docker volumes
func UseDockerVolumes() bool {
	// Проверяем переменную окружения
	if env := os.Getenv("USE_DOCKER_VOLUMES"); env != "" {
		return env == "true"
	}

	// Автоопределение по платформе
	switch DetectPlatform() {
	case PlatformMac:
		return true // На Mac используем volumes (нет File Sharing)
	case PlatformLinux:
		return false // На Linux используем bind mounts
	case PlatformWindows:
		return true // На Windows используем volumes (проще)
	default:
		return true // По умолчанию volumes
	}
}

// GetMountSource возвращает источник для монтирования
func GetMountSource(studentID string) (string, error) {
	log := logger.Get()

	if UseDockerVolumes() {
		// Режим 1: Docker volume (работает на Mac и Windows)
		volumeName := fmt.Sprintf("student-data-%s", studentID)

		// Проверяем существование volume
		checkCmd := exec.Command("docker", "volume", "inspect", volumeName)
		if err := checkCmd.Run(); err != nil {
			log.Info("Volume %s will be created automatically", volumeName)
		}

		return volumeName, nil
	} else {
		// Режим 2: Bind mount (для Linux)
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("cannot get current directory: %v", err)
		}

		// На Linux путь должен быть абсолютным
		bindPath := filepath.Join(cwd, "students", studentID, "work")

		// Создаем директорию
		if err := os.MkdirAll(bindPath, 0755); err != nil {
			return "", fmt.Errorf("failed to create work dir: %v", err)
		}

		log.Info("Using bind mount: %s", bindPath)
		return bindPath, nil
	}
}

// FindFreePort находит свободный порт
func FindFreePort(basePort int) (int, error) {
	log := logger.Get()
	log.Info("Searching for free port starting from %d", basePort)

	for port := basePort; port < basePort+1000; port++ {
		// Проверяем Docker контейнеры
		checkCmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("publish=%d", port), "-q")
		if out, _ := checkCmd.Output(); len(out) > 0 {
			log.Debug("Port %d is used by another container", port)
			continue
		}

		// Проверяем доступность порта на хосте
		if isPortAvailable(port) {
			log.Info("Found free port: %d", port)
			return port, nil
		}

		log.Debug("Port %d is not available", port)
	}

	return 0, fmt.Errorf("no free ports found in range %d-%d", basePort, basePort+1000)
}

// isPortAvailable проверяет доступность порта
func isPortAvailable(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	ln.Close()
	time.Sleep(50 * time.Millisecond)
	return true
}

// Start runs Docker container with VNC desktop
func Start(basePath, studentID string) error {
	log := logger.Get()
	log.LogEvent("START_ATTEMPT", studentID, "starting", nil)

	image := os.Getenv("LAB_IMAGE")
	if image == "" {
		image = config.LabImage
	}

	// Получаем источник для монтирования
	mountSource, err := GetMountSource(studentID)
	if err != nil {
		log.Error("Failed to get mount source: %v", err)
		return err
	}

	log.Info("Platform: %s, Using volumes: %v, Mount source: %s",
		DetectPlatform(), UseDockerVolumes(), mountSource)

	// Находим свободный порт
	webPort, err := FindFreePort(10000)
	if err != nil {
		log.Error("Failed to find free port: %v", err)
		return fmt.Errorf("failed to find free port: %v", err)
	}

	containerName := ContainerName(studentID)

	// Удаляем старый контейнер
	existing, _ := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", containerName), "-q").Output()
	if len(existing) > 0 {
		log.Warn("Removing existing container %s", containerName)
		exec.Command("docker", "rm", "-f", containerName).Run()
	}

	// Формируем аргументы для монтирования
	var mountArg string
	if UseDockerVolumes() {
		mountArg = fmt.Sprintf("%s:/home/ubuntu/work", mountSource)
	} else {
		mountArg = fmt.Sprintf("%s:/home/ubuntu/work", mountSource)
	}

	// Запускаем контейнер
	args := []string{
		"run", "-d",
		"--name", containerName,
		"--restart", "no",
		"-p", fmt.Sprintf("%d:80", webPort),
		"-v", mountArg,
		"-e", fmt.Sprintf("VNC_PW=%s", config.VNCPassword),
		"--memory", "512m",
		"--cpus", "0.5",
		image,
	}

	// Добавляем platform для Mac (чтобы избежать предупреждений)
	if DetectPlatform() == PlatformMac {
		args = append([]string{"--platform", "linux/amd64"}, args...)
	}

	log.Info("Running: docker %v", args)
	cmd := exec.Command("docker", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("Docker run failed: %v\n%s", err, string(output))
		return fmt.Errorf("docker run failed: %v", err)
	}

	// Ждем запуска
	time.Sleep(2 * time.Second)

	// Проверяем, что контейнер запущен
	running, err := IsRunning(studentID)
	if err != nil || !running {
		log.Error("Container %s is not running", containerName)
		return fmt.Errorf("container failed to start")
	}

	log.Info("Container %s started on port %d", containerName, webPort)
	log.LogEvent("START_SUCCESS", studentID, "running", map[string]string{
		"port":         fmt.Sprintf("%d", webPort),
		"mount_source": mountSource,
	})

	return nil
}

// Stop stops and removes the container
func Stop(studentID string) error {
	log := logger.Get()
	containerName := ContainerName(studentID)

	out, _ := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", containerName), "-q").Output()
	if len(out) == 0 {
		log.Warn("Container %s does not exist", containerName)
		return nil
	}

	log.Info("Stopping container %s", containerName)
	exec.Command("docker", "stop", "--time", "10", containerName).Run()
	exec.Command("docker", "rm", containerName).Run()

	log.LogEvent("STOP_SUCCESS", studentID, "stopped", nil)
	return nil
}

// Info returns URL for the lab
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

// IsRunning checks if container is running
func IsRunning(studentID string) (bool, error) {
	containerName := ContainerName(studentID)
	out, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", containerName).Output()
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(string(out)) == "true", nil
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

// WorkDirPath returns absolute path to student work directory on host
// Для обратной совместимости с существующим кодом
func WorkDirPath(basePath, studentID string) (string, error) {
	log := logger.Get()

	// Используем GetMountSource для получения правильного пути
	mountSource, err := GetMountSource(studentID)
	if err != nil {
		log.Error("Failed to get mount source in WorkDirPath: %v", err)
		return "", err
	}

	// Если это volume, возвращаем путь внутри контейнера (для информации)
	if UseDockerVolumes() {
		// Для volumes нет прямого пути на хосте
		// Возвращаем путь в контейнере для информации
		return fmt.Sprintf("/app/students/%s/work (Docker volume)", studentID), nil
	}

	// Для bind mount возвращаем реальный путь
	return mountSource, nil
}
