# Виртуальная образовательная лаборатория (Go)

Дипломный проект: бэкенд на Go вместо bash-скрипта. Запуск Docker-лабораторий (VNC), бэкапы и восстановление данных студентов.

## Требования

- Go 1.21+
- Docker (для запуска лабораторий)

## Сборка

```bash
go build -o edu-lab ./cmd/edu-lab
```

## CLI (как в оригинальном скрипте)

```bash
# Запустить лабораторию для студента
./edu-lab start student1

# Остановить и сделать бэкап
./edu-lab stop student1

# Тест бэкапов (без Docker)
./edu-lab test

# Тест лаборатории (Docker + VNC)
./edu-lab test-lab
```

После `start` открывайте **http://localhost:8080**, пароль VNC: `vncpassword`. Рабочая папка в контейнере: `/home/ubuntu/work` (она примонтирована из `students/<ИМЯ>/work/`).

## HTTP API

Запуск сервера:

```bash
./edu-lab -server
# по умолчанию порт 9000 (лаборатория заняла 8080)
./edu-lab -server -port=9000
```

Эндпоинты (POST, JSON):

| Метод | URL | Тело |
|-------|-----|------|
| POST | `/api/structure` | — | Создать каталоги `backups/`, `students/` |
| POST | `/api/start` | `{"student_id": "student1"}` | Запустить лабораторию |
| POST | `/api/stop` | `{"student_id": "student1"}` | Остановить и сделать бэкап |
| POST | `/api/backup` | `{"student_id": "student1"}` | Только создать бэкап |
| POST | `/api/restore` | `{"student_id": "student1", "backup_file": "backups/student1_20260215_120000.tar.gz"}` | Восстановить из бэкапа |

Пример:

```bash
curl -X POST http://localhost:9000/api/start \
  -H "Content-Type: application/json" \
  -d '{"student_id":"student1"}'
```

## Структура проекта

```
edu-lab-platform/
├── cmd/edu-lab/main.go   # CLI и запуск HTTP-сервера
├── internal/
│   ├── config/           # Константы (каталоги, порт, образ Docker)
│   ├── backup/           # Создание/восстановление/очистка бэкапов
│   ├── lab/              # Запуск/остановка Docker-контейнера
│   └── server/           # HTTP API
├── go.mod
├── backups/              # создаётся при работе
└── students/             # данные студентов (work — монтируется в контейнер)
```

Оригинальный bash-скрипт можно сохранить рядом (например `lab.sh`) и при необходимости вызывать его; весь функционал перенесён в Go.
