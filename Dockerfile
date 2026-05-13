# Сборка React (Vite)
FROM node:20-alpine AS webbuilder
WORKDIR /web
COPY web/package.json ./
RUN npm install --no-audit --no-fund
COPY web/ ./
RUN npm run build

# Сборка Go
FROM golang:1.25-alpine AS gobuilder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=webbuilder /web/dist ./web/dist
RUN CGO_ENABLED=0 go build -o edu-lab ./cmd/edu-lab

# Финальный образ
FROM alpine:3.19

RUN apk add --no-cache \
    docker \
    docker-cli \
    docker-compose \
    bash \
    curl \
    lsof \
    && rm -rf /var/cache/apk/*

RUN mkdir -p /app/logs /app/students /app/backups /app/web/dist

COPY --from=gobuilder /app/edu-lab /usr/local/bin/
COPY --from=webbuilder /web/dist /app/web/dist

WORKDIR /app
EXPOSE 9000

ENTRYPOINT ["edu-lab"]
CMD ["-server", "-port=9000"]
