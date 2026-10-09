-- Заявки клиентов с сайта (контактные формы и т.п.).
-- Создание заявки публикует событие в Kafka (topic "leads"), которое читает telegram-bot
-- и присылает уведомление владельцу/сотрудникам.

CREATE TABLE leads (
    id BIGSERIAL PRIMARY KEY,
    service_id BIGINT REFERENCES services(id) ON DELETE SET NULL,
    name VARCHAR(100) NOT NULL,
    phone TEXT NOT NULL CHECK (phone ~ '^\+[1-9][0-9]{6,14}$'),
    email VARCHAR(255),
    message VARCHAR(5000),
    source VARCHAR(100) NOT NULL DEFAULT 'contact_form',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX leads_created_at_idx ON leads (created_at DESC);
CREATE INDEX leads_service_id_idx ON leads (service_id);
