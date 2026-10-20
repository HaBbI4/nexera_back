# Frontend Audit — Nexera

Дата аудита: 2026-10-20  
Источник истины: `nexera_back/` (Go HTTP API, PostgreSQL, Redis).

## Existing architecture

- **Монорепозиторий workspace:** `nexera/` содержит только backend `nexera_back/`. Публичного frontend в репозитории нет.
- **Backend:** Go 1.26, стандартный `net/http` + `ServeMux` (без Gin/Fiber/ORM), слои `handlers → services → repository → PostgreSQL`.
- **Инфра:** PostgreSQL, Redis (кеш + rate limit), Kafka/outbox заложены, но **не подключены** к server/telegram-bot (уведомления о лидах пока не уходят в Telegram).
- **Админка:** JWT (`/api/v1/admin/*`), роли (`admin` и др.), CRUD контента.
- **Медиа:** local/S3 для изображений баннеров, раздача `/media/...`.
- **Документация API:** Swagger UI на `/swagger/`, контракт форм — `nexera_back/docs/frontend-forms.md`.
- **Git:** отдельный remote `https://github.com/HaBbI4/nexera_back.git` внутри `nexera_back/`. Корневой workspace git не инициализирован.
- **Тесты:** unit-тесты сервисов на fake repositories; middleware CORS/rate-limit покрыты тестами.

## Available backend capabilities

| Возможность | Публичный API | Админ API | Примечание |
|---|---|---|---|
| Баннеры | `GET /api/v1/banners`, `/{id}` | полный CRUD + upload | Фильтр по `page_id`, окно видимости |
| Страницы | `GET /api/v1/pages`, `/{id}` | CRUD | Слаг (`home`, `about`…), баннеры при GET by id |
| Услуги | `GET /api/v1/services`, `/{id}` | CRUD | `name`, `description`, `price`, `position` |
| FAQ | `GET /api/v1/faq`, `/{id}` | CRUD | |
| Отзывы | `GET /api/v1/reviews`, `/summary`, `POST` | модерация | Публично только approved, без контактов |
| Заявки (лиды) | `POST /api/v1/leads` | list/get/delete | Honeypot `website`, rate limit, duplicate phone |
| Health | `/health`, readiness, liveness | — | |
| Contact info | **нет handler** | **нет** | Таблица `contact_info` + model есть, API отсутствует |
| Кейсы / портфолио | **нет** | **нет** | Сущности нет в миграциях |
| Загрузка файлов к заявке | **нет** | — | Только media для баннеров (admin) |

### Контракт заявки (`POST /api/v1/leads`)

Обязательные: `name`, `phone` (E.164, допускаются пробелы/скобки/дефисы).  
Опциональные: `email`, `message`, `service_id`, `source` (по умолчанию `contact_form`).  
Honeypot: `website` (пустое).  
Ответы: `201` успех (в т.ч. silent для spam), `400`, `429`, `5xx`.  
**Важно:** отправлять из браузера (не SSR), иначе общий IP лимита — см. `frontend-forms.md`.

### Реальные бизнес-данные в коде

- Бренд: **Nexera** (модуль `nexera_back`, описание «IT-компании / сайт-визитка»).
- Контакт поддержки API в swagger: `911gtrss@gmail.com` — **не** подтверждённый публичный контакт компании для сайта.
- Seed услуг/кейсов/контактов/FAQ в миграциях **отсутствует** — контент ожидается через админку.
- Подтверждённых кейсов, отзывов, метрик, адреса, телефона в репозитории **нет**.

## API integration map

| Frontend need | Backend source | Strategy |
|---|---|---|
| Hero / баннеры главной | `GET /pages` + banners by `page_id` или `GET /banners?page_id=` | SSR fetch + fallback copy |
| Каталог услуг | `GET /services` | SSR; детальные SEO-страницы по slug из локального mapping + id |
| FAQ | `GET /faq` | SSR; если пусто — аккуратные общие ответы без выдуманных условий |
| Отзывы | `GET /reviews` + `/summary` | Показывать блок только если `count > 0` |
| Быстрая форма / бриф | `POST /leads` | Client-side fetch, `source`: `quick_contact` / `project_brief` / `case_cta` |
| Контакты | **нужен** `GET /api/v1/contact-info` | Добавить публичный (и admin CRUD) endpoint |
| Кейсы | нет API | Пустое состояние + `CONTENT_REQUIRED.md`; структура страниц готова |
| Политика ПДн | нет | Статическая страница-заглушка + CONTENT_REQUIRED |

## Missing capabilities

1. **Публичный/админ API для `contact_info`** — таблица есть, handlers/repository/service нет.
2. **Сущность кейсов (portfolio)** — нет модели, миграций, медиа-галереи.
3. **Slug у услуг** — только числовой `id`; для SEO нужны либо миграция `slug`, либо клиентский slug map.
4. **Файловые вложения к заявкам** — не поддерживать до появления безопасного storage + API.
5. **Kafka → Telegram уведомления о лидах** — не подключены (не блокер для сохранения заявок в БД).
6. **Публичный frontend** — отсутствует целиком.
7. **Контент:** услуги, FAQ, контакты, кейсы, отзывы, политика — заполняются вручную / через админку.

## Proposed frontend architecture

- **Стек:** Next.js App Router + TypeScript + Tailwind CSS + Motion (`motion/react`) + React Hook Form + Zod.
- **Расположение:** `nexera_web/` рядом с `nexera_back/` (расширение workspace, не greenfield-rewrite backend).
- **Данные:** серверные fetch для SEO-страниц (`services`, `faq`, `banners`, `reviews`); формы — только client `fetch` на `/api/v1/leads` через Next rewrite/proxy на backend.
- **Конфиг:** `NEXT_PUBLIC_SITE_URL`, `NEXT_PUBLIC_API_BASE` (пусто = same-origin proxy), централизованный `lib/site.ts` для бренда/навигации.
- **Контент без API:** `content/` (process, positioning, budget ranges для брифа) — редактируемые константы, без фейковых кейсов/отзывов.
- **Design system:** CSS variables, expressive fonts (не Inter/system), graphite + porcelain + electric accent.
- **21st.dev:** MCP-сервер 21st в сессии Cursor **недоступен**; CLI `21st` доступен (`search`/`get`). AI generate выключен (`aiGenerationEnabled: false`) — используем search→get→адаптация.

## Integration and security risks

| Риск | Митигация |
|---|---|
| SSR POST лидов → общий IP rate limit | Только browser fetch; Next rewrite `/api/*` → backend |
| Honeypot / silent 201 | На любой 201 показывать успех; не полагаться на тело |
| CORS `*` | Допустимо (Bearer, не cookies); prod лучше same-origin через nginx |
| Секреты в клиенте | Только public env; JWT/admin не в marketing site |
| Пустые услуги/FAQ из БД | Graceful empty + статичный fallback FAQ без ложных обещаний |
| Нет контактов в API | Добавить contact-info API; UI скрывает пустые каналы |
| Выдуманный social proof | Не показывать отзывы/метрики/кейсы без данных |
| `prefers-reduced-motion` | Все motion-эффекты уважают reduced motion |

## Reference (web-valley.ru)

Изучена структура (бизнес-референс, **не** копирование дизайна): сильный hero + CTA, портфолио, процесс, многошаговый бриф, контакты. Nexera позиционируется шире (приложения/SaaS/продукты), визуально — собственная editorial/engineering айдентика.

## 21st.dev status (this machine)

- MCP namespace `21st`: после reconnect пользователя **по-прежнему отсутствует** в agent tool catalog (`GetDynamicTools` → только `cursor` / `cursor-cloud` / `cursor-subscriptions`). Конфиг есть в `~/.cursor/mcp.json` и `.cursor/mcp.json`.
- CLI: **доступен**, login `gama17one`, paid tier, **AI generation disabled** — использованы `21st search` / `21st get` для всех ключевых блоков §4.
- Детальный лог: `docs/21ST_COMPONENTS.md`.

## Decision summary

1. Не переписывать backend — добавить только недостающий **contact-info** public/admin API по существующим паттернам.
2. Создать `nexera_web` (Next.js) как production marketing site.
3. Кейсы: маршруты + empty state; без вымышленных проектов.
4. Лиды: реальный `POST /api/v1/leads` с `source` analytics.
