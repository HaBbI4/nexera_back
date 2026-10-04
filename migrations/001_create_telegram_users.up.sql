-- Пользователи telegram-бота (владелец и сотрудники, получающие уведомления и доступ к боту).
-- Первая запись добавляется вручную после раскатки миграции; в дальнейшем управляется через админку.
CREATE TABLE telegram_users (
    id BIGSERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Один Telegram-чат не может быть добавлен дважды
CREATE UNIQUE INDEX idx_telegram_users_chat_id
    ON telegram_users(chat_id);
