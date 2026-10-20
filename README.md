# Nexera

Маркетинговый сайт студии цифровых продуктов + backend сайта-визитки.

## Структура

| Путь | Назначение |
|---|---|
| `nexera_back/` | Go API, Postgres, Redis, админка, заявки |
| `nexera_back/web/` | Next.js App Router — публичный сайт |
| `docs/` | Аудит фронтенда, контент-дыры, 21st-отчёт |

## Быстрый старт

### 1. Backend

См. подробности в `nexera_back/CLAUDE.md`.

```bash
cd nexera_back
cp .env.example .env   # задать JWT_SECRET
make dev-up            # postgres + redis
make migrate-up
make run               # :8080
```

### 2. Frontend

```bash
cd nexera_back/web
cp .env.example .env.local
npm install
npm run dev            # :3000, /api/* → localhost:8080
```

Переменные:

- `NEXT_PUBLIC_SITE_URL` — канонический URL сайта
- `API_INTERNAL_URL` — куда Next проксирует `/api` и `/media` (по умолчанию `http://localhost:8080`)

### Сборка фронтенда

```bash
cd nexera_back/web
npm run build
npm start
```

## Формы заявок

- Быстрый контакт и бриф → `POST /api/v1/leads` из браузера (honeypot `website`, rate limit).
- `source`: `quick_contact` | `project_brief` | `contact_form` | `case_cta`
- Инструкция: `nexera_back/docs/frontend-forms.md`

## Документы

- `docs/FRONTEND_AUDIT.md` — аудит backend / план интеграции
- `docs/CONTENT_REQUIRED.md` — что заполнить вручную
- `docs/21ST_COMPONENTS.md` — компоненты 21st.dev (CLI; MCP в agent session не был виден)

## Дизайн

Направление **Signal Forge**: graphite + porcelain + electric mint, шрифты Syne / Outfit.
