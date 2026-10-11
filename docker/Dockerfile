# Build stage
FROM golang:1.26-alpine AS builder

# Установка зависимостей для сборки
RUN apk add --no-cache git ca-certificates

# Создание рабочей директории
WORKDIR /build

# Копирование go.mod и go.sum для кеширования зависимостей
COPY go.mod go.sum ./

# Загрузка зависимостей
RUN go mod download

# Копирование исходного кода
COPY . .

# Сборка приложения
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o nexera_back-server ./cmd/server
# Служебные команды (create-admin и т.п.): docker compose run --rm app ./nexera_back-cli ...
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o nexera_back-cli ./cmd/cli

# Production stage
FROM alpine:latest

# Установка необходимых утилит
RUN apk --no-cache add ca-certificates tzdata wget

# Создание пользователя для безопасности
RUN adduser -D -s /bin/sh appuser

# Создание рабочей директории
WORKDIR /app

# Создание директорий для логов и локально хранимых медиа-файлов
RUN mkdir -p /app/logs /app/uploads && chown -R appuser:appuser /app/logs /app/uploads

# Копирование бинарного файла из builder stage
COPY --from=builder /build/nexera_back-server .
COPY --from=builder /build/nexera_back-cli .

# Изменение владельца файлов
RUN chown appuser:appuser nexera_back-server nexera_back-cli

# Переключение на пользователя appuser
USER appuser

# Экспорт порта
EXPOSE 8080

# Healthcheck
# Обычный net/http.ServeMux матчит "GET /health" строго по методу, поэтому wget --spider
# (шлёт HEAD) всегда получал бы 405 — используем реальный GET.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 -O /dev/null http://localhost:8080/health || exit 1

# Запуск приложения
CMD ["./nexera_back-server"]
