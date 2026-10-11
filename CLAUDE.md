# Nexera Back — сайт-визитка с баннерами, заявками и Telegram-уведомлениями

## 📋 Содержание

- [Описание проекта](#описание-проекта)
- [Функционал](#функционал)
- [Архитектура](#архитектура)
- [Быстрый старт](#быстрый-старт)
- [Мониторинг](#мониторинг)
- [Разработка](#разработка)

## 🎯 Описание проекта

Находится в разработке.

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
2. **Middleware**: `internal/middleware` — CORS (`CORS` на каждом маршруте добавляет заголовки ответа; preflight `OPTIONS` обрабатывает `Preflight`, обернутый вокруг всего `ServeMux`: маршруты зарегистрированы с методом в паттерне, и без него `ServeMux` ответил бы на `OPTIONS` 405), структурированное логирование запросов, JWT-аутентификация (`RequireAuth`), проверка ролей (`RequireRole`)
3. **Handlers**: разделены на `internal/handlers/admin` (требуют JWT) и `internal/handlers/public` (без авторизации); общие вещи (health, media) — в `internal/handlers`
4. **Auth**: `internal/auth` — выпуск/проверка JWT, bcrypt для паролей
5. **Services**: `internal/services` — бизнес-логика, оркестрация, не содержит SQL
6. **Repository**: `internal/repository` — единственный слой, обращающийся к БД напрямую; каждый репозиторий — интерфейс + реализация на `database/sql`, что и позволяет писать unit-тесты сервисов без поднятия Postgres
7. **Data Layer**: PostgreSQL — основное хранилище, Redis — кеш
8. **Event System**: Kafka + transactional outbox — асинхронные уведомления для telegram-bot
9. **Monitoring**: health checks и структурированные логи (`internal/logger`, JSON/text через logrus)

> Уведомления через Kafka + outbox пока не реализованы: пакет `internal/kafka` (`Producer.Publish`, `Consumer`, топик `KAFKA_TOPIC_EVENTS`) и таблица `outbox_events` заложены, но ни `cmd/server`, ни `cmd/telegram-bot` к Kafka сейчас не подключаются. Бот пока только проверяет доступ по `telegram_users` и отвечает на `/start`.

## 🚀 Быстрый старт

### 1. Поднять инфраструктуру (Postgres, Redis) и накатить миграции

```bash
make dev-up
make migrate-up
```

### 2. Создать администратора

Роль `admin` создается миграцией `010_seed_admin_role`, самого пользователя — CLI-командой (аналог `php artisan`, исходники в `cmd/cli`):

```bash
make create-admin email=admin@example.com name="Администратор"
# пароль спрашивается в терминале; без терминала — через ADMIN_PASSWORD
```

В Docker: `docker compose run --rm app ./nexera_back-cli create-admin --email admin@example.com`. Пароль в `.env` не храним: команда запускается один раз, дальше пароль меняется через админку.

### 3. Запустить сервер локально

```bash
make run
# или напрямую: go run cmd/server/main.go
```

Обязательно должен быть задан `JWT_SECRET` в `.env` — без него сервер не стартует (см. `.env.example`).

### 4. Проверка работоспособности

```bash
make health
# или: curl http://localhost:8080/health
```

### 5. Полный стек в Docker (app + telegram-bot + postgres + redis + kafka)

```bash
make up
make logs
```

`docker-compose.yml` не хранит секреты захардкоженными — `DB_USER`/`DB_PASSWORD`/`DB_NAME` и `JWT_SECRET` подставляются во все сервисы (`postgres`, `migrate`, `app`, `telegram-bot`) из `.env` через интерполяцию (`${DB_USER:-nexera_back_user}` и т.п.), а не прописаны в compose-файле напрямую. `JWT_SECRET` обязателен без дефолта: если он не задан в `.env`, `docker compose up` откажет ещё на этапе интерполяции с понятным сообщением — раньше без него `app` уходил в бесконечный рестарт-луп (`restart: always` + `log.Fatal` при пустом `JWT_SECRET`).

**Важно:** в Docker-образе лежит только скомпилированный бинарник (multi-stage build, финальный слой — `alpine` без исходников и без hot-reload тулов). Правки в коде **не применяются** к уже запущенному контейнеру — ни простой `docker compose restart app`, ничего: нужно пересобрать образ (`docker compose up -d --build app`). Для активной разработки используйте `make dev-up` (только postgres+redis в Docker) + `make run` (сервер из исходников на хосте, `go run` подхватывает правки при каждом перезапуске) — `make up` поднимает весь стек целиком, это ближе к прод-подобному прогону, а не к циклу разработки.

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
│   ├── cli/               # Служебные команды (create-admin)
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
6. **Ошибки "не найдено"/"невалидные данные"**: сигнальные ошибки (`var ErrXNotFound = errors.New(...)` в `repository`, `var ErrInvalidX = errors.New(...)` в `services`, конкретные причины через `fmt.Errorf("%w: ...", ErrInvalidX)`), в хендлере — `errors.Is(err, ...)` для выбора HTTP-статуса. Не разбор текста ошибки через `strings.Contains` — см. `internal/repository/banner_repository.go` / `internal/services/banner_service.go` / `internal/handlers/admin/banner_handler.go`

### Добавление новой сущности (новый домен)

1. Модель в `internal/models/` (один файл на сущность, с `CreateXRequest` при необходимости)
2. Интерфейс + Postgres-реализация в `internal/repository/x_repository.go`
3. Бизнес-логика в `internal/services/x_service.go`, зависит от интерфейса репозитория, не от `*sql.DB`
4. Unit-тесты сервиса на фейковом репозитории (`internal/services/x_service_test.go`)
5. HTTP-обработчик в `internal/handlers/admin/` (если требует JWT) или `internal/handlers/public/`
6. Маршрут в `cmd/server/main.go`, с нужным набором middleware (`middleware.Chain(handler, middleware.Logging(log), middleware.CORS, ...)`)

За конкретными решениями (фильтрация/сортировка списка, частичное обновление, обработка ошибок) — см. следующий раздел: баннеры задуманы как эталон для всего проекта, а не разовое решение только для одной сущности.

### Эталонный паттерн CRUD-эндпоинтов (баннеры)

`internal/repository/banner_repository.go` + `internal/services/banner_service.go` + `internal/handlers/public/banner_handler.go` + `internal/handlers/admin/banner_handler.go` — образец, на который стоит ориентироваться для всех новых сущностей (услуги, FAQ, контакты, лиды, отзывы).

**Чтение — `GET /api/v1/banners` (список) и `GET /api/v1/banners/{id}`:**
- Фильтрация/сортировка по белому списку: `BannerSortField`/`bannerSortColumns` в репозитории — колонка в `ORDER BY` никогда не строится из сырого ввода клиента (SQL-инъекция через идентификатор колонки иначе неизбежна, т.к. `$N`-плейсхолдеры работают только для значений, не для имён колонок/направления)
- Бизнес-правило видимости (`is_active` + окно `starts_at`/`ends_at`) живёт в сервисе (`ListPublic`/`GetPublicByID` всегда форсируют `OnlyVisible: true`), а не в хендлере — его нельзя обойти, выбрав другой путь вызова сервиса
- Админке скрытое тоже нужно видеть, поэтому у сущностей с флагом видимости (`is_active`/`is_published`) есть отдельные `GET /api/v1/admin/x` и `/api/v1/admin/x/{id}` под `RequireAuth`: сервис вызывает `List`/`GetByID` без форсирования видимости, а флаг можно отфильтровать параметром (`is_active`/`is_published`). Один публичный GET с необязательной авторизацией не делаем: ответ одного URL зависел бы от того, кто спрашивает, а скрытый объект мог бы попасть в публичный Redis-кеш. Админские GET не кешируются
- Если в сущности есть персональные данные клиента (телефон, email — например, отзывы), публичные эндпоинты отдают отдельный тип без этих полей (`models.PublicReview`, получается через `Review.ToPublic()` в сервисе), а не обнуляют поля полной модели в хендлере: у публичного типа просто нет полей, через которые контакты могли бы утечь. Полная модель — только в `handlers/admin`
- Валидация query-параметров в хендлере **не единообразна по степени строгости специально**: `limit`/`offset` при некорректном значении тихо откатываются к дефолту, а `page_id`/`sort`/`order` — новый для проекта вид параметров, там некорректное значение это `400`, а не тихий дефолт, чтобы клиент не думал, что получил отсортированный список, а на деле получил дефолтную сортировку

**Запись — `POST /api/v1/admin/banners`, `PATCH /api/v1/admin/banners/{id}`, `DELETE /api/v1/admin/banners/{id}`:**
- `PATCH` не собирает частичный `UPDATE` в репозитории — сервис подгружает текущую сущность (`repo.GetByID`), накладывает поверх только присланные в запросе поля (`nil` в `models.UpdateBannerRequest` = не менять), валидирует получившееся целиком и только потом сохраняет полную перезапись через `repo.Update(banner)`. Благодаря этому кросс-полевые правила (например, `ends_at` должен быть позже `starts_at`) проверяются корректно, даже если в конкретном PATCH-запросе менялось только одно из двух полей
- `validateBanner` — одна функция на `Create` и `Update`: бизнес-правило обязано выполняться независимо от того, как сущность получила текущий вид (целиком при создании или слиянием при обновлении)
- Нарушение внешнего ключа (`page_id` не существует) репозиторий ловит через `pq.Error.Code == "23503"` и транслирует в сигнальную `ErrPageNotFound` — иначе клиент увидел бы сырую ошибку Postgres вместо `400 page not found`
- `UpdateBannerRequest` с `nil`-полями не умеет явно сбросить необязательное поле обратно в `NULL` (только задать новое значение) — осознанное упрощение для сайта-визитки, не стоит решать double-pointer'ами, пока не появится реальная потребность
- Запись инвалидирует Redis-кеш конкретного ID после Update/Delete (`redisClient.Delete(...)`), а не ждёт TTL

### Защита публичных форм от спама

Публичные `POST` без авторизации (`/api/v1/leads`, `/api/v1/reviews`) защищены тремя слоями; новые публичные формы подключаются так же:

1. **Лимит запросов с IP** — `middleware.RateLimit` (фиксированное окно, счетчик `INCR` + `EXPIRE NX` в Redis, нужен Redis 7+). Сверх лимита — `429` + `Retry-After`. Если Redis недоступен, запрос пропускается (fail-open): потерять заявку хуже, чем пропустить спам
2. **Honeypot** — скрытое поле формы `website`. Заполнено — сервис возвращает `services.ErrSpamDetected`, хендлер отвечает `201` без сохранения, чтобы бот не понял, что его отсекли. Фронтенд обязан отправлять это поле пустым (или не отправлять)
3. **Повтор по телефону** — сервис отклоняет заявку/отзыв, если с тем же телефоном уже была запись в пределах окна (`ErrDuplicateSubmission` → `429`). Для отзывов учитываются и удаленные

На лимит и на повтор ответ одинаковый (`429`, `httpresponse.TooManyRequestsMessage`): клиенту не сообщаем, какая проверка сработала и что заявка с этим телефоном уже есть. Все лимиты и окна — в `.env` (`*_RATE_LIMIT_*`, `*_DUPLICATE_WINDOW_MINUTES`), `0` отключает проверку.

**IP клиента** — только через `middleware.ClientIPResolver`, не `r.RemoteAddr` и не сырой `X-Forwarded-For`: за nginx `RemoteAddr` — адрес nginx (у всех посетителей был бы общий лимит), а `X-Forwarded-For` клиент может подделать. Заголовку верим только от прокси из `TRUSTED_PROXIES` (аналог `TrustProxies` в Laravel), читаем справа налево до первого недоверенного адреса. nginx должен передавать заголовок: `proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;`

Что нужно делать фронтенду (отправка из браузера, а не с SSR-сервера, поле `website`, обработка `201`/`429`) — `docs/frontend-forms.md`; при изменении правил защиты обновлять и его.

### Аутентификация админ-панели

JWT выдаётся через `POST /api/v1/admin/auth/login` (`internal/handlers/admin/auth_handler.go`), проверяется `middleware.RequireAuth(jwtSecret)` на защищённых маршрутах. Роль пользователя — в `middleware.RequireRole(codes...)`, читается из `middleware.UserFromContext(ctx)`. Пароли — только bcrypt (`internal/auth/password.go`), в JSON-ответах `PasswordHash` всегда обнуляется перед сериализацией.

### Миграции БД

Для добавления новой миграции:

1. Создайте файлы `XXX_name.up.sql` и `XXX_name.down.sql` в папке `migrations/`
2. `make migrate-create name=add_something` создаст пару файлов автоматически
3. Единый стиль: `NOW()` заглавными, без явного `NULL` для nullable-колонок, суффиксы индексов `_idx`/`_uq`
