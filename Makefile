# Переменные
BINARY_NAME=nexera_back-server
DOCKER_IMAGE=nexera_back
VERSION=latest

# Go команды
.PHONY: build run swagger docker-build docker-run up down logs dev-up dev-down health migrate-up migrate-down migrate-create create-admin

# Сборка бинарника
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) cmd/server/main.go

# Запуск приложения с переменными из .env (сам сервер .env не читает)
run:
	@set -a; . ./.env; set +a; \
	go run cmd/server/main.go

# Генерация Swagger/OpenAPI документации из аннотаций в коде.
# После изменения @Summary/@Param/@Router в internal/handlers обязательно запускать заново.
swagger:
	@swag init -g cmd/server/main.go -o internal/docs --parseDependency --parseInternal

# Docker команды

# Сборка образа приложения
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE):$(VERSION) -f docker/Dockerfile .

# Запуск одного контейнера с приложением (без postgres/redis)
docker-run:
	@echo "Running Docker container..."
	@docker run -p 8080:8080 $(DOCKER_IMAGE):$(VERSION)

# Docker Compose команды

# Поднять весь стек: postgres, redis, миграции, приложение
up:
	@echo "Starting all services..."
	@docker compose up -d --build

# Остановить весь стек
down:
	@echo "Stopping all services..."
	@docker compose down

# Логи всех сервисов
logs:
	@echo "Showing logs..."
	@docker compose logs -f

# Поднять только инфраструктуру (postgres, redis) для локальной разработки,
# приложение при этом запускается локально через `make run`
dev-up:
	@echo "Starting development environment..."
	@docker compose up -d postgres redis

# Остановить окружение разработки
dev-down:
	@echo "Stopping development environment..."
	@docker compose down

# Проверка состояния приложения
health:
	@echo "Checking application health..."
	@curl -s http://localhost:8080/health | jq

# Миграции БД (локально, напрямую на локальный Postgres из .env).
# Требует установленный CLI: brew install golang-migrate (или go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest)

# Накатить все непримененные миграции
migrate-up:
	@set -a; . ./.env; set +a; \
	migrate -path migrations -database "postgres://$${DB_USER}:$${DB_PASSWORD}@$${DB_HOST}:$${DB_PORT}/$${DB_NAME}?sslmode=$${DB_SSL_MODE}" up

# Откатить последнюю миграцию
migrate-down:
	@set -a; . ./.env; set +a; \
	migrate -path migrations -database "postgres://$${DB_USER}:$${DB_PASSWORD}@$${DB_HOST}:$${DB_PORT}/$${DB_NAME}?sslmode=$${DB_SSL_MODE}" down 1

# Создать новую пару файлов миграции: make migrate-create name=add_something
migrate-create:
	@migrate create -seq -ext sql -dir migrations $(name)

# Создать администратора админ-панели (роль admin заводит миграция 010): make create-admin email=admin@example.com [name="Имя"]
# Пароль спрашивается в терминале; для неинтерактивного запуска — переменная ADMIN_PASSWORD.
create-admin:
	@set -a; . ./.env; set +a; \
	go run ./cmd/cli create-admin --email "$(email)" $(if $(name),--name "$(name)")
