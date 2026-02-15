# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Копируем go.mod и go.sum (если есть)
COPY go.mod ./
# COPY go.sum ./

# Скачиваем зависимости
RUN go mod download

# Копируем весь исходный код
COPY . .

# Собираем приложение
RUN go build -o edu-lab ./cmd/edu-lab

# Финальный образ
FROM alpine:latest

# Устанавливаем необходимые пакеты
RUN apk add --no-cache \
    docker \
    docker-cli \
    docker-compose \
    bash \
    curl \
    lsof \
    && rm -rf /var/cache/apk/*

# Создаем необходимые директории
RUN mkdir -p /app/logs \
    /app/students \
    /app/backups \
    /app/web

# Копируем скомпилированное приложение
COPY --from=builder /app/edu-lab /usr/local/bin/

# Копируем веб-интерфейс
COPY web/index.html /app/web/

# Устанавливаем рабочую директорию
WORKDIR /app

# Открываем порты
EXPOSE 9000 8080

# Добавляем пользователя
RUN adduser -D -h /app labuser && \
    chown -R labuser:labuser /app

USER labuser

ENTRYPOINT ["edu-lab"]
CMD ["-server", "-port=9000"]