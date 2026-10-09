# Nexera Back — сайт-визитка с баннерами, заявками и Telegram-уведомлениями

## 📋 Содержание

- [Описание проекта](#описание-проекта)
- [Функционал](#функционал)
- [Архитектура](#архитектура)
- [Быстрый старт](#быстрый-старт)
- [Мониторинг](#мониторинг)
- [Разработка](#разработка)

## 🎯 Описание проекта

Находится в разработке (форк от `my-blog`, детейлинг-домен полностью вытесняется).

Backend для сайта-визитки IT-компании: баннеры с фильтрацией по страницам, каталог услуг, FAQ, контакты, отзывы клиентов и приём заявок (лидов). Отдельно — Telegram-бот для владельца/сотрудников: получает уведомления о новых заявках и отзывах через Kafka.

- **Управлять контентом сайта**: баннеры, услуги, FAQ, контакты — через админ-панель
- **Принимать обращения клиентов**: заявки (лиды) и отзывы с модерацией
- **Обрабатывать события**: лиды/отзывы публикуются в Kafka через transactional outbox, читаются telegram-bot
- **Кешировать данные**: быстрый доступ через Redis
- **Мониторить состояние**: health checks и структурированное логирование

## 💡 Функционал

### 1. Публичная часть сайта

Баннеры (привязаны к странице через `page_id`, порядок — полем `position`), каталог услуг, FAQ, контакты — отдаются на чтение без авторизации. Заявки и отзывы клиенты создают сами, без авторизации; отзыв не показывается на сайте, пока его не одобрит админ (`is_approved`).

### 2. Админ-панель

Вход по email/паролю (JWT), роли пользователей (`users`/`roles`). CRUD для баннеров/услуг/FAQ/контактов, просмотр и модерация заявок/отзывов.

### 3. Телеграм-бот

Уведомляет владельца/сотрудников о новых заявках и отзывах. Доступ к боту ограничен таблицей `telegram_users` (отдельно от JWT-авторизации админки). Читает события из Kafka, которые API публикует через transactional outbox (`outbox_events`) при создании лида/отзыва.

## 🏗 Архитектура

Слоистая архитектура на стандартной библиотеке Go (без веб-фреймворка и ORM):

```
HTTP-запрос
    │
    ▼
middleware (CORS → Logging → RequireAuth/RequireRole)
    │
    ▼
handlers/admin  или  handlers/public   (разбор запроса, вызов сервиса, формирование ответа)
    │
    ▼
services        (бизнес-логика, не знает про SQL)
    │
    ▼
repository       (SQL-запросы, скрыты за интерфейсом — в тестах подменяются фейком)
    │
    ▼
database (*sql.DB) → PostgreSQL
```

Kafka (`internal/kafka`) и Redis (`internal/redis`) подключаются к этому стеку сбоку: Kafka — для событий лидов/отзывов (читает `cmd/telegram-bot`), Redis — для кеширования горячих данных на уровне handlers.

### Компоненты

1. **HTTP API**: RESTful API без внешних фреймворков (`net/http`, `http.ServeMux` с Go 1.22+ паттернами маршрутов)
2. **Middleware**: `internal/middleware` — CORS, структурированное логирование запросов, JWT-аутентификация (`RequireAuth`), проверка ролей (`RequireRole`)
3. **Handlers**: разделены на `internal/handlers/admin` (требуют JWT) и `internal/handlers/public` (без авторизации); общие вещи (health, media) — в `internal/handlers`
4. **Auth**: `internal/auth` — выпуск/проверка JWT, bcrypt для паролей
5. **Services**: `internal/services` — бизнес-логика, оркестрация, не содержит SQL
6. **Repository**: `internal/repository` — единственный слой, обращающийся к БД напрямую; каждый репозиторий — интерфейс + реализация на `database/sql`, что и позволяет писать unit-тесты сервисов без поднятия Postgres
7. **Data Layer**: PostgreSQL — основное хранилище, Redis — кеш
8. **Event System**: Kafka + transactional outbox — асинхронные уведомления для telegram-bot
9. **Monitoring**: health checks и структурированные логи (`internal/logger`, JSON/text через logrus)

> Старый детейлинг-домен (`cars`, `customers`, `orders`, `service_service.go` и их хендлеры в `internal/handlers`) ещё не вынесен — он донашивается временно и будет заменён хендлерами/сервисами/репозиториями под баннеры/лиды/отзывы по той же схеме, что уже сделана для `users` (см. `internal/repository/user_repository.go`, `internal/services/user_service.go`, `internal/handlers/admin/auth_handler.go`).

## 🚀 Быстрый старт

### 1. Поднять инфраструктуру (Postgres, Redis) и накатить миграции

```bash
make dev-up
make migrate-up
```

### 2. Запустить сервер локально

```bash
make run
# или напрямую: go run cmd/server/main.go
```

Обязательно должен быть задан `JWT_SECRET` в `.env` — без него сервер не стартует (см. `.env.example`).

### 3. Проверка работоспособности

```bash
make health
# или: curl http://localhost:8080/health
```

### 4. Полный стек в Docker (app + telegram-bot + postgres + redis + kafka)

```bash
make up
make logs
```

## 📊 Мониторинг

### Health Checks

- `/health` - полная проверка здоровья всех компонентов
- `/health/readiness` - готовность к обслуживанию запросов
- `/health/liveness` - жизнеспособность приложения

### Логирование запросов

Все HTTP-запросы логируются через `middleware.Logging` (метод, путь, статус-код, длительность) структурированным логгером (`internal/logger`, JSON или text — настраивается через `LOG_FORMAT`).

## 👨‍💻 Разработка

### Структура проекта

```
nexera_back/
├── cmd/
│   ├── server/            # HTTP API
│   └── telegram-bot/      # Телеграм-бот (читает Kafka, шлёт уведомления)
├── internal/
│   ├── auth/              # JWT (выпуск/проверка), bcrypt
│   ├── config/            # Конфигурация из переменных окружения
│   ├── database/          # Подключение к PostgreSQL (*sql.DB)
│   ├── handlers/
│   │   ├── admin/         # Хендлеры, требующие JWT (middleware.RequireAuth)
│   │   └── public/        # Хендлеры без авторизации (лиды, отзывы, баннеры и т.д.)
│   ├── httpresponse/       # Общие хелперы JSON-ответов (WriteJSON/WriteError)
│   ├── kafka/              # Kafka producer/consumer
│   ├── logger/             # Структурированное логирование
│   ├── media/              # Хранилище медиа (local/s3)
│   ├── middleware/         # CORS, Logging, RequireAuth, RequireRole
│   ├── models/              # Структуры данных (по одной сущности на файл)
│   ├── redis/               # Redis клиент
│   ├── repository/          # SQL-запросы за интерфейсами, по сущности на файл
│   ├── services/             # Бизнес-логика поверх repository
│   └── telegram/             # Telegram-бот (telebot.v3)
├── migrations/               # SQL миграции (golang-migrate)
├── docker/                   # Dockerfile для app и telegram-bot
├── docker-compose.yml
├── Makefile
├── go.mod
└── CLAUDE.md
```

### Соглашения по коду

1. **Именование**: camelCase для переменных, PascalCase для типов
2. **Обработка ошибок**: всегда обрабатывайте ошибки явно, оборачивайте через `fmt.Errorf("...: %w", err)`
3. **Логирование**: структурированные логи с контекстом (`log.WithFields(...)`), не `fmt.Printf`
4. **Тесты**: сервисы тестируются через фейковые реализации `repository.*Repository` — см. `internal/services/user_service_test.go` как образец. Покрытие бизнес-логики — приоритет, 80% не является твёрдым требованием на данном этапе, но стремимся к нему по мере переноса доменов
5. **Слои**: handlers не обращаются к БД напрямую; вся работа с SQL — только через `internal/repository`

### Добавление новой сущности (новый домен)

1. Модель в `internal/models/` (один файл на сущность, с `CreateXRequest` при необходимости)
2. Интерфейс + Postgres-реализация в `internal/repository/x_repository.go`
3. Бизнес-логика в `internal/services/x_service.go`, зависит от интерфейса репозитория, не от `*sql.DB`
4. Unit-тесты сервиса на фейковом репозитории (`internal/services/x_service_test.go`)
5. HTTP-обработчик в `internal/handlers/admin/` (если требует JWT) или `internal/handlers/public/`
6. Маршрут в `cmd/server/main.go`, с нужным набором middleware (`middleware.Chain(handler, middleware.Logging(log), middleware.CORS, ...)`)

### Аутентификация админ-панели

JWT выдаётся через `POST /api/v1/admin/auth/login` (`internal/handlers/admin/auth_handler.go`), проверяется `middleware.RequireAuth(jwtSecret)` на защищённых маршрутах. Роль пользователя — в `middleware.RequireRole(codes...)`, читается из `middleware.UserFromContext(ctx)`. Пароли — только bcrypt (`internal/auth/password.go`), в JSON-ответах `PasswordHash` всегда обнуляется перед сериализацией.

### Миграции БД

Для добавления новой миграции:

1. Создайте файлы `XXX_name.up.sql` и `XXX_name.down.sql` в папке `migrations/`
2. `make migrate-create name=add_something` создаст пару файлов автоматически
3. Единый стиль: `NOW()` заглавными, без явного `NULL` для nullable-колонок, суффиксы индексов `_idx`/`_uq`
