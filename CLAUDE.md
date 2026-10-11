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

Два режима запуска:

- **Полный стек в Docker** (`app` + `telegram-bot` + postgres + redis + kafka). Ближе к продакшену, подходит для ручного тестирования API.
- **Локальная разработка.** В Docker только postgres и redis, сервер запускается из исходников через `go run`.

### Полный стек в Docker

#### 0. Что нужно заранее

- Запущенный Docker Desktop.
- Свободные порты 8080, 5433, 6379, 9092, 29092 и 2181.
- `jq` по желанию, для удобного вывода JSON: `brew install jq`.

#### 1. Подготовить `.env`

```bash
cp .env.example .env      # если .env еще нет
openssl rand -hex 32      # сгенерировать значение для JWT_SECRET
```

- **`JWT_SECRET`** обязателен. Без него `docker compose up` откажет еще на этапе интерполяции.
- **`TELEGRAM_BOT_TOKEN`.** Чтобы проверить бота, получите токен у [@BotFather](https://t.me/BotFather) (`/newbot`). Без валидного токена контейнер `telegram-bot` будет падать и перезапускаться по кругу, поэтому без бота стек поднимайте командой из шага 2.

`docker-compose.yml` не хранит секреты: `DB_USER`/`DB_PASSWORD`/`DB_NAME` и `JWT_SECRET` подставляются во все сервисы (`postgres`, `migrate`, `app`, `telegram-bot`) из `.env` через интерполяцию (`${DB_USER:-nexera_back_user}` и т.п.). У `JWT_SECRET` нет дефолта специально: раньше без него `app` уходил в бесконечный рестарт-луп (`restart: always` + `log.Fatal` при пустом `JWT_SECRET`). Остальные настройки `app` (хосты БД/Redis/Kafka внутри docker-сети, лимиты и т.п.) заданы в compose-файле.

#### 2. Поднять стек

```bash
make up                              # всё, включая telegram-bot (нужен TELEGRAM_BOT_TOKEN)
docker compose up -d --build app     # без бота: app и его зависимости
```

- **Миграции применяются автоматически.** Сервис `migrate` выполняет `migrations/` до старта `app`, включая `010_seed_admin_role`, которая создает роль `admin`.
- **Kafka сервер пока не использует**, но она указана в `depends_on`. Поэтому первый старт занимает 1–2 минуты, пока Kafka не станет healthy.

#### 3. Проверить, что всё поднялось

```bash
docker compose ps                    # app — Up (healthy), migrate — Exited (0)
docker compose logs migrate          # последняя применённая миграция — 10/u seed_admin_role
curl -s localhost:8080/health | jq   # статус всех компонентов
docker compose logs -f app           # если app не стартует
```

#### 4. Создать администратора

Роль `admin` создает миграция, а пользователя — CLI-команда (аналог `php artisan`, исходники в `cmd/cli`). Пароль команда спрашивает в терминале дважды (минимум 8 символов). Для неинтерактивного запуска пароль передается через `ADMIN_PASSWORD`.

```bash
# внутри запущенного контейнера
docker compose exec app ./nexera_back-cli create-admin --email admin@example.com --name "Админ"

# или с хоста: .env смотрит на Postgres из Docker (localhost:5433)
make create-admin email=admin@example.com name="Админ"
```

Пароль в `.env` не храним: команда запускается один раз, дальше пароль меняется через админку.

#### 5. Залогиниться и получить JWT

```bash
TOKEN=$(curl -s -X POST localhost:8080/api/v1/admin/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"ВАШ_ПАРОЛЬ"}' | jq -r .token)

curl -s localhost:8080/api/v1/admin/auth/me -H "Authorization: Bearer $TOKEN" | jq
```

- **Все `/api/v1/admin/*`** требуют заголовок `Authorization: Bearer $TOKEN`. В Postman это вкладка Authorization, тип Bearer Token.
- **Время жизни токена** задает `JWT_TTL_MINUTES`, по умолчанию 24 часа.

#### 6. Что тестировать

- **Swagger UI:** http://localhost:8080/swagger/index.html, там все эндпоинты и форматы запросов. Авторизация по Bearer в Swagger пока не настроена, поэтому админские запросы отправляйте через curl или Postman.
- **Публичные эндпоинты (без токена):**
  - `GET /api/v1/banners`, `/pages`, `/services`, `/faq`.
  - `POST /api/v1/leads` и `/api/v1/reviews` с защитой от спама (см. «Защита публичных форм от спама»). Например, 6-я заявка с одного IP за 10 минут получит `429`.
- **Админские эндпоинты:**
  - CRUD баннеров, страниц, услуг, FAQ;
  - просмотр лидов и модерация отзывов;
  - только для роли `admin`: пользователи, роли, `telegram-users`.
- **Загрузка изображений баннеров.** Файл передается полем `image` в multipart-запросе:
  ```bash
  curl -s -X POST localhost:8080/api/v1/admin/banners -H "Authorization: Bearer $TOKEN" \
    -F page_id=1 -F title="Тест" -F image=@/путь/к/картинке.jpg | jq
  ```
  - В ответе `image_url` — относительный путь вида `/media/banners/<файл>.jpg`, картинка открывается по `http://localhost:8080` + `image_url`.
  - Файлы лежат в томе `app-uploads`: `docker compose exec app ls /app/uploads/banners`.
  - Полная ссылка вместо относительной включается через `MEDIA_LOCAL_PUBLIC_BASE_URL`.

#### 7. Проверить telegram-бота (если запущен с токеном)

1. Узнать свой chat_id: написать [@userinfobot](https://t.me/userinfobot).
2. Добавить себя в бота:
   ```bash
   curl -s -X POST localhost:8080/api/v1/admin/telegram-users \
     -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
     -d '{"chat_id": 123456789, "name": "Я"}' | jq
   ```
3. Написать боту `/start`, он ответит приветствием. Если записи нет или у нее `is_active=false`, доступ запрещен. Уведомлений о заявках пока нет (см. примечание про outbox выше).

#### 8. Полезные команды

```bash
make logs                                  # логи всех сервисов
docker compose logs -f app                 # только API
docker compose up -d --build app           # пересобрать после правок кода
docker compose exec postgres psql -U nexera_back_user -d nexera_back_db   # консоль БД
make down                                  # остановить стек
docker compose down -v                     # остановить и удалить все данные (БД, загрузки)
```

После `down -v` база пустая: при следующем `up` миграции применятся заново, а администратора нужно создать снова (шаг 4).

**Важно:** в Docker-образе лежат только скомпилированные бинарники: multi-stage build, финальный слой — `alpine` без исходников и без hot-reload тулов. Правки в коде **не применяются** к уже запущенному контейнеру, `docker compose restart app` тоже не поможет — нужно пересобрать образ: `docker compose up -d --build app`.

### Локальная разработка

```bash
make dev-up                                       # postgres + redis в Docker
make migrate-up                                   # нужен CLI golang-migrate: brew install golang-migrate
make create-admin email=admin@example.com         # один раз
make run                                          # сервер из исходников на :8080
make health                                       # или: curl http://localhost:8080/health
```

- **`make run` и `make create-admin`** подгружают переменные из `.env`, сам сервер `.env` не читает. Если запускаете напрямую через `go run cmd/server/main.go`, переменные нужно экспортировать самому.
- **`go run` подхватывает правки** при каждом перезапуске, поэтому этот режим удобнее для цикла разработки, чем пересборка образа.

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
