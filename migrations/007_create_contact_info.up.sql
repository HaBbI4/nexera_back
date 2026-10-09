-- Контактные данные компании (телефон, email, мессенджеры и т.д.), отображаемые на сайте.
CREATE TABLE contact_info (
    id BIGSERIAL PRIMARY KEY,
    type TEXT NOT NULL CHECK (type IN ('phone','email','telegram','whatsapp','vk')),
    value VARCHAR(100),
    label VARCHAR(100),
    position INTEGER NOT NULL DEFAULT 0 CHECK (position >= 0),
    is_published BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);