package models

import (
	"time"

	"github.com/google/uuid"
)

// EventType представляет тип события, публикуемого в Kafka
type EventType string

const (
	// EventTypeOrderCreated публикуется при создании нового заказа
	EventTypeOrderCreated EventType = "order.created"
	// EventTypeOrderStatusChanged публикуется при смене статуса заказа
	EventTypeOrderStatusChanged EventType = "order.status_changed"
)

// Event представляет конверт события Kafka. ID уникален для конкретного события
// (не путать с ID заказа) и используется как ключ сообщения и для идемпотентности на стороне читателя.
type Event struct {
	ID        uuid.UUID   `json:"id"`
	Type      EventType   `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// OrderCreatedEvent представляет событие создания заказа.
// Намеренно не несет полные данные по заказу (услуги, фото и т.д.) — читатель события
// (например, telegram-bot) обращается за деталями к БД по OrderID.
type OrderCreatedEvent struct {
	OrderID      int       `json:"order_id"`
	CustomerName string    `json:"customer_name"`
	CarBrand     string    `json:"car_brand"`
	CarModel     string    `json:"car_model"`
	TotalAmount  float64   `json:"total_amount"`
	CreatedAt    time.Time `json:"created_at"`
}

// OrderStatusChangedEvent представляет событие изменения статуса заказа
type OrderStatusChangedEvent struct {
	OrderID   int         `json:"order_id"`
	OldStatus OrderStatus `json:"old_status"`
	NewStatus OrderStatus `json:"new_status"`
	Timestamp time.Time   `json:"timestamp"`
}
