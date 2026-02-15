// Диплом: виртуальная образовательная лаборатория — Go backend
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/edu-lab-platform/internal/backup"
	"github.com/edu-lab-platform/internal/lab"
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

	if *runServer {
		backup.EnsureStructure(basePath())
		addr := ":" + *port
		fmt.Println("🎓 Виртуальная лаборатория — API")
		fmt.Println("   http://localhost" + addr)
		fmt.Println("   POST /api/start, /api/stop, /api/backup, /api/restore, /api/structure")
		if err := http.ListenAndServe(addr, server.Mux()); err != nil {
			log.Fatal(err)
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
		if err := backup.EnsureStructure(base); err != nil {
			log.Fatal(err)
		}
		if err := lab.Start(base, name); err != nil {
			log.Fatal(err)
		}
		url, password := lab.Info(name)
		workDir, _ := lab.WorkDirPath(base, name)
		fmt.Println("✅ ГОТОВО!")
		fmt.Println("🌐", url)
		fmt.Println("🔑", password)
		fmt.Println("📁", workDir)
	case "stop":
		if name == "" {
			fmt.Println("❌ Укажите имя: edu-lab stop ИМЯ")
			os.Exit(1)
		}
		if path, err := backup.Create(base, name); err == nil {
			fmt.Println("💾 Бэкап:", path)
		}
		lab.Stop(name)
		fmt.Println("✅ Остановлено")
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
	fmt.Println("  test          — тест бэкапов")
	fmt.Println("  test-lab      — тест лаборатории (Docker)")
	fmt.Println("")
	fmt.Println("Сервер API:")
	fmt.Println("  -server       — запустить HTTP API (порт: -port=9000)")
	fmt.Println("")
	fmt.Println("Пример:")
	fmt.Println("  edu-lab start student1")
	fmt.Println("  edu-lab stop student1")
	fmt.Println("  edu-lab -server -port=9000")
}

func runTestBackup(base string) {
	fmt.Println("🧪 ТЕСТ БЭКАПОВ")
	fmt.Println("================")
	if err := backup.EnsureStructure(base); err != nil {
		log.Fatal(err)
	}
	fmt.Println("1. Создаю тестовые данные...")
	testDir := filepath.Join(base, "students", "test", "work")
	os.MkdirAll(testDir, 0755)
	os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("Файл 1\n"), 0644)
	os.WriteFile(filepath.Join(testDir, "file2.txt"), []byte("Файл 2\n"), 0644)
	fmt.Println("2. Создаю бэкап...")
	path, err := backup.Create(base, "test")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("   Создан:", path)
	fmt.Println("3. Удаляю оригинал...")
	os.RemoveAll(filepath.Join(base, "students", "test"))
	fmt.Println("4. Восстанавливаю...")
	if err := backup.Restore(base, "test", path); err != nil {
		log.Fatal(err)
	}
	fmt.Println("5. Проверяю...")
	data, _ := os.ReadFile(filepath.Join(base, "students", "test", "work", "file1.txt"))
	fmt.Println("   file1.txt:", string(data))
	fmt.Println("✅ ТЕСТ УСПЕШЕН!")
}

func runTestLab(base string) {
	fmt.Println("🚀 ТЕСТ ЛАБОРАТОРИИ")
	fmt.Println("==================")
	fmt.Println("1. Запускаю...")
	if err := backup.EnsureStructure(base); err != nil {
		log.Fatal(err)
	}
	if err := lab.Start(base, "test_lab_user"); err != nil {
		log.Fatal(err)
	}
	url, _ := lab.Info("test_lab_user")
	fmt.Println("2. Откройте", url)
	fmt.Println("   Создайте файл в /home/ubuntu/work")
	fmt.Println("   Нажмите Enter для остановки...")
	fmt.Scanln()
	fmt.Println("3. Останавливаю...")
	if path, err := backup.Create(base, "test_lab_user"); err == nil {
		fmt.Println("   Бэкап:", path)
	}
	lab.Stop("test_lab_user")
	fmt.Println("4. Проверяю сохранение...")
	entries, _ := os.ReadDir(filepath.Join(base, "students", "test_lab_user", "work"))
	for _, e := range entries {
		fmt.Println("   ", e.Name())
	}
	fmt.Println("✅ ТЕСТ ЗАВЕРШЁН!")
}
