-- Отзывы клиентов. Публикуются на сайте только после модерации (is_approved).
-- Создание отзыва публикует событие в Kafka (topic "reviews") для уведомления в telegram-bot.
CREATE TABLE reviews (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    phone TEXT NOT NULL CHECK (phone ~ '^\+[1-9][0-9]{6,14}$'),
    email VARCHAR(100),
    rating SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    text TEXT NOT NULL,
    is_approved BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX reviews_is_approved_idx ON reviews (is_approved);
CREATE INDEX reviews_created_at_idx ON reviews (created_at DESC);
