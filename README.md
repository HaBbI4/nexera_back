# Nexera Back

Backend сайта-визитки IT-компании: баннеры, услуги, FAQ, контакты, отзывы, заявки (лиды), JWT-админка, Telegram-бот (в разработке).

Публичный маркетинговый сайт вынесен в отдельный проект: **[`nexera_front`](https://github.com/gama1726/nexera_front)**
(локально: `../nexera_front`).

## Быстрый старт

См. подробности в `CLAUDE.md`.

```bash
cp .env.example .env   # задать JWT_SECRET; при необходимости CORS_ALLOWED_ORIGINS
make dev-up            # postgres + redis
make migrate-up
make run               # :8080
```

### CORS для nexera_front

Фронтенд на отдельном origin (например `http://localhost:3000`). Разрешённые Origin:

```env
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
# production:
# CORS_ALLOWED_ORIGINS=http://localhost:3000,https://your-site.example
```

`*` — разрешить любой Origin (прежнее поведение).

## Публичные формы

- `POST /api/v1/leads` — заявки с сайта
- `GET /api/v1/contact-info` — контакты для витрины
- Инструкция для фронта: `docs/frontend-forms.md` (копия также в `nexera_front/docs/`)

## Документы (backend)

- `CLAUDE.md` — архитектура и соглашения
- `docs/frontend-forms.md` — контракт лидов/отзывов
- Аудит и контент маркетингового сайта — в **`nexera_front/docs/`**
