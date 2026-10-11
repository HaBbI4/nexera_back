package models

import (
	"time"

	"github.com/google/uuid"
)

// EventType представляет тип события, публикуемого в Kafka (например, "lead.created").
// Константы типов заводятся вместе с событиями, когда появится transactional outbox.
type EventType string

// Event представляет конверт события Kafka. ID уникален для конкретного события
// (не путать с ID сущности) и используется как ключ сообщения и для идемпотентности на стороне читателя.
type Event struct {
	ID        uuid.UUID   `json:"id"`
	Type      EventType   `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}
