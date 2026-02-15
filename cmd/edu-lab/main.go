// Диплом: виртуальная образовательная лаборатория — Go backend
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/edu-lab-platform/internal/backup"
	"github.com/edu-lab-platform/internal/lab"
	"github.com/edu-lab-platform/internal/logger"
	"github.com/edu-lab-platform/internal/server"
)

func basePath() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}

func main() {
	runServer := flag.Bool("server", false, "запустить HTTP API сервер")
	port := flag.String("port", "9000", "порт HTTP сервера (при -server); лаборатория использует 8080")
	flag.Parse()

	// Инициализация логгера
	logDir := filepath.Join(basePath(), "logs")
	if err := logger.Init(logDir); err != nil {
		fmt.Printf("⚠️ Предупреждение: не удалось инициализировать логгер: %v\n", err)
	}

	// Получаем экземпляр логгера
	log := logger.Get()
	defer log.Close()

	if *runServer {
		if err := backup.EnsureStructure(basePath()); err != nil {
			log.Error("Ошибка создания структуры: %v", err)
			fmt.Printf("❌ Ошибка: %v\n", err)
			os.Exit(1)
		}

		addr := ":" + *port

		log.Info("🚀 Запуск HTTP API сервера на порту %s", *port)
		fmt.Println("🎓 Виртуальная лаборатория — API")
		fmt.Println("   http://localhost" + addr)
		fmt.Println("   POST /api/start, /api/stop, /api/backup, /api/restore, /api/structure")
		fmt.Println("   GET /api/status, /api/list, /api/logs")

		if err := http.ListenAndServe(addr, server.Mux()); err != nil {
			log.Error("❌ Сервер остановлен с ошибкой: %v", err)
			fmt.Printf("❌ Ошибка сервера: %v\n", err)
			os.Exit(1)
		}
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		return
	}

	cmd, name := args[0], ""
	if len(args) > 1 {
		name = args[1]
	}

	base := basePath()

	switch cmd {
	case "start":
		if name == "" {
			fmt.Println("❌ Укажите имя: edu-lab start ИМЯ")
			os.Exit(1)
		}

		log.LogEvent("START_ATTEMPT", name, "начинаю", nil)

		if err := backup.EnsureStructure(base); err != nil {
			log.Error("Ошибка создания структуры: %v", err)
			fmt.Printf("❌ Ошибка: %v\n", err)
			os.Exit(1)
		}

		if err := lab.Start(base, name); err != nil {
			log.Error("Ошибка запуска лаборатории для %s: %v", name, err)
			fmt.Printf("❌ Ошибка запуска: %v\n", err)
			os.Exit(1)
		}

		url, password := lab.Info(name)
		workDir, _ := lab.WorkDirPath(base, name)

		log.LogEvent("START_SUCCESS", name, "успешно", map[string]string{
			"url":      url,
			"work_dir": workDir,
			"password": password,
		})

		fmt.Println("✅ ГОТОВО!")
		fmt.Println("🌐", url)
		fmt.Println("🔑", password)
		fmt.Println("📁", workDir)

	case "stop":
		if name == "" {
			fmt.Println("❌ Укажите имя: edu-lab stop ИМЯ")
			os.Exit(1)
		}

		log.LogEvent("STOP_ATTEMPT", name, "начинаю", nil)

		var backupPath string
		if path, err := backup.Create(base, name); err == nil {
			backupPath = path
			fmt.Println("💾 Бэкап:", path)
			log.Info("Бэкап создан: %s", path)
		} else {
			log.Warn("Не удалось создать бэкап для %s: %v", name, err)
		}

		if err := lab.Stop(name); err != nil {
			log.Warn("Ошибка при остановке %s: %v", name, err)
		}

		log.LogEvent("STOP_SUCCESS", name, "остановлено", map[string]string{
			"backup": backupPath,
		})

		fmt.Println("✅ Остановлено")

	case "status":
		showStatus(base)

	case "list":
		listStudents(base)

	case "test":
		runTestBackup(base)

	case "test-lab":
		runTestLab(base)

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("🎓 ВИРТУАЛЬНАЯ ЛАБОРАТОРИЯ (Go)")
	fmt.Println("")
	fmt.Println("Команды:")
	fmt.Println("  start ИМЯ     — запустить лабораторию")
	fmt.Println("  stop ИМЯ      — остановить и сделать бэкап")
	fmt.Println("  status        — показать статус системы")
	fmt.Println("  list          — список всех студентов")
	fmt.Println("  test          — тест бэкапов")
	fmt.Println("  test-lab      — тест лаборатории (Docker)")
	fmt.Println("")
	fmt.Println("Сервер API:")
	fmt.Println("  -server       — запустить HTTP API (порт: -port=9000)")
	fmt.Println("")
	fmt.Println("Пример:")
	fmt.Println("  edu-lab start student1")
	fmt.Println("  edu-lab stop student1")
	fmt.Println("  edu-lab status")
	fmt.Println("  edu-lab -server -port=9000")
}

func showStatus(base string) {
	log := logger.Get()

	fmt.Println("📊 СТАТУС СИСТЕМЫ")
	fmt.Println("================")

	// Проверка Docker
	dockerCheck := exec.Command("docker", "info")
	if err := dockerCheck.Run(); err != nil {
		fmt.Println("🐳 Docker: ❌ НЕ ЗАПУЩЕН")
		log.Warn("Docker не запущен")
	} else {
		fmt.Println("🐳 Docker: ✅ РАБОТАЕТ")
	}

	// Активные контейнеры
	containers, err := lab.GetAllRunning()
	if err != nil {
		fmt.Printf("❌ Ошибка получения списка контейнеров: %v\n", err)
	} else {
		fmt.Printf("📦 Активных лабораторий: %d\n", len(containers))
		for _, c := range containers {
			fmt.Println("   -", c)
		}
	}

	// Статистика по папкам
	studentsDir := filepath.Join(base, "students")
	backupsDir := filepath.Join(base, "backups")
	logsDir := filepath.Join(base, "logs")

	studentCount := countDirs(studentsDir)
	backupCount := countFiles(backupsDir, "*.tar.gz")

	fmt.Printf("👥 Студентов: %d\n", studentCount)
	fmt.Printf("💾 Бэкапов: %d\n", backupCount)

	// Размер на диске
	fmt.Println("\n💽 Использование диска:")
	printDirSize(studentsDir, "   students/")
	printDirSize(backupsDir, "   backups/")
	printDirSize(logsDir, "   logs/")
}

func listStudents(base string) {
	fmt.Println("👥 СПИСОК СТУДЕНТОВ")
	fmt.Println("==================")

	studentsDir := filepath.Join(base, "students")
	entries, err := os.ReadDir(studentsDir)
	if err != nil {
		fmt.Println("❌ Нет данных о студентах или папка не существует")
		return
	}

	found := false
	for _, entry := range entries {
		if entry.IsDir() {
			found = true
			studentID := entry.Name()
			running, _ := lab.IsRunning(studentID)

			status := "⏹️  Остановлен"
			if running {
				url, _ := lab.Info(studentID)
				status = fmt.Sprintf("🟢 Работает (%s)", url)
			}

			fmt.Printf("   %s - %s\n", studentID, status)

			// Считаем бэкапы
			backupPattern := filepath.Join(base, "backups", studentID+"_*.tar.gz")
			backups, _ := filepath.Glob(backupPattern)
			if len(backups) > 0 {
				fmt.Printf("      📦 Бэкапов: %d\n", len(backups))
			}
		}
	}

	if !found {
		fmt.Println("   Нет студентов")
	}
}

func runTestBackup(base string) {
	log := logger.Get()
	log.Info("🧪 Запуск теста бэкапов")

	fmt.Println("🧪 ТЕСТ БЭКАПОВ")
	fmt.Println("================")

	if err := backup.EnsureStructure(base); err != nil {
		log.Error("Ошибка создания структуры: %v", err)
		fmt.Printf("❌ Ошибка: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("1. Создаю тестовые данные...")
	testDir := filepath.Join(base, "students", "test", "work")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		log.Error("Ошибка создания тестовой папки: %v", err)
		fmt.Printf("❌ Ошибка: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("Файл 1\n"), 0644); err != nil {
		log.Error("Ошибка записи тестового файла: %v", err)
		fmt.Printf("❌ Ошибка: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(filepath.Join(testDir, "file2.txt"), []byte("Файл 2\n"), 0644); err != nil {
		log.Error("Ошибка записи тестового файла: %v", err)
		fmt.Printf("❌ Ошибка: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("2. Создаю бэкап...")
	path, err := backup.Create(base, "test")
	if err != nil {
		log.Error("Ошибка создания бэкапа: %v", err)
		fmt.Printf("❌ Ошибка: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("   Создан:", path)

	fmt.Println("3. Удаляю оригинал...")
	if err := os.RemoveAll(filepath.Join(base, "students", "test")); err != nil {
		log.Error("Ошибка удаления: %v", err)
		fmt.Printf("❌ Ошибка: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("4. Восстанавливаю...")
	if err := backup.Restore(base, "test", path); err != nil {
		log.Error("Ошибка восстановления: %v", err)
		fmt.Printf("❌ Ошибка: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("5. Проверяю...")
	data, err := os.ReadFile(filepath.Join(base, "students", "test", "work", "file1.txt"))
	if err != nil {
		log.Error("Ошибка чтения восстановленного файла: %v", err)
		fmt.Printf("❌ Ошибка: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("   file1.txt:", string(data))

	log.Info("✅ Тест бэкапов успешно завершен")
	fmt.Println("✅ ТЕСТ УСПЕШЕН!")
}

func runTestLab(base string) {
	log := logger.Get()
	log.Info("🚀 Запуск теста лаборатории")

	fmt.Println("🚀 ТЕСТ ЛАБОРАТОРИИ")
	fmt.Println("==================")

	fmt.Println("1. Запускаю...")
	if err := backup.EnsureStructure(base); err != nil {
		log.Error("Ошибка создания структуры: %v", err)
		fmt.Printf("❌ Ошибка: %v\n", err)
		os.Exit(1)
	}

	if err := lab.Start(base, "test_lab_user"); err != nil {
		log.Error("Ошибка запуска: %v", err)
		fmt.Printf("❌ Ошибка: %v\n", err)
		os.Exit(1)
	}

	url, _ := lab.Info("test_lab_user")
	fmt.Println("2. Откройте", url)
	fmt.Println("   Создайте файл в /home/ubuntu/work")
	fmt.Println("   Нажмите Enter для остановки...")
	fmt.Scanln()

	fmt.Println("3. Останавливаю...")
	if path, err := backup.Create(base, "test_lab_user"); err == nil {
		fmt.Println("   Бэкап:", path)
		log.Info("Бэкап создан: %s", path)
	}

	if err := lab.Stop("test_lab_user"); err != nil {
		log.Warn("Ошибка при остановке: %v", err)
	}

	fmt.Println("4. Проверяю сохранение...")
	entries, err := os.ReadDir(filepath.Join(base, "students", "test_lab_user", "work"))
	if err != nil {
		fmt.Println("   Нет файлов")
	} else {
		for _, e := range entries {
			fmt.Println("   ", e.Name())
		}
	}

	log.Info("✅ Тест лаборатории успешно завершен")
	fmt.Println("✅ ТЕСТ ЗАВЕРШЁН!")
}

// Вспомогательные функции
func countDirs(path string) int {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			count++
		}
	}
	return count
}

func countFiles(path, pattern string) int {
	matches, err := filepath.Glob(filepath.Join(path, pattern))
	if err != nil {
		return 0
	}
	return len(matches)
}

func printDirSize(path, prefix string) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	if err != nil || size == 0 {
		fmt.Printf("%s0 B\n", prefix)
	} else if size < 1024*1024 {
		fmt.Printf("%s%.2f KB\n", prefix, float64(size)/1024)
	} else {
		fmt.Printf("%s%.2f MB\n", prefix, float64(size)/1024/1024)
	}
}
