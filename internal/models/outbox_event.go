package models

import (
	"encoding/json"
	"time"
)

// OutboxEvent представляет запись transactional outbox: событие для надежной публикации в Kafka.
// Запись в БД и запись события происходят в одной транзакции, что гарантирует доставку
// даже при сбое producer'а.
type OutboxEvent struct {
	ID          int             `json:"id" db:"id"`
	Payload     json.RawMessage `json:"payload" db:"payload"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	PublishedAt *time.Time      `json:"published_at,omitempty" db:"published_at"`

	Topic     string `json:"topic" db:"topic"`
	Key       string `json:"key" db:"key"`
	EventType string `json:"event_type" db:"event_type"`
}
