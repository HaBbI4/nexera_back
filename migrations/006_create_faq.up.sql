-- Вопросы и ответы (FAQ) для страниц сайта.
CREATE TABLE faq (
    id BIGSERIAL PRIMARY KEY,
    question VARCHAR(255) NOT NULL,
    answer VARCHAR(5000) NOT NULL,
    position INTEGER NOT NULL DEFAULT 0 CHECK (position >= 0),
    is_published BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX faq_published_position_idx
    ON faq (position, id) WHERE is_published;