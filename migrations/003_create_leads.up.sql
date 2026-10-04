-- Заявки клиентов с сайта (контактные формы и т.п.).
-- Создание заявки публикует событие в Kafka (topic "leads"), которое читает telegram-bot
-- и присылает уведомление владельцу/сотрудникам.
CREATE TYPE lead_status AS ENUM ('new', 'in_progress', 'done', 'rejected');

CREATE TABLE leads (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    email VARCHAR(255),
    message TEXT,
    source VARCHAR(100) NOT NULL DEFAULT 'contact_form',
    status lead_status NOT NULL DEFAULT 'new',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_leads_status ON leads(status);
CREATE INDEX idx_leads_created_at ON leads(created_at DESC);
