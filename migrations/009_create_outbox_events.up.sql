-- Transactional outbox: события для надежной публикации в Kafka (гарантирует доставку
-- даже при сбое producer'а — запись в БД и запись события происходят в одной транзакции).
CREATE TABLE outbox_events (
    id BIGSERIAL PRIMARY KEY,
    topic TEXT NOT NULL,
    key TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
);

CREATE INDEX outbox_unpublished_idx ON outbox_events (id) WHERE published_at IS NULL;
