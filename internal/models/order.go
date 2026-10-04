package models

import "time"

// OrderStatus представляет статусы заказы
type OrderStatus string

const (
	OrderStatusCreated    OrderStatus = "created"
	OrderStatusScheduled  OrderStatus = "scheduled"
	OrderStatusInProgress OrderStatus = "in_progress"
	OrderStatusCompleted  OrderStatus = "completed"
	OrderStatusCancelled  OrderStatus = "cancelled"
)

// Order Представляет запись клиента на обслуживание
type Order struct {
	ID          int     `json:"id" db:"id"`
	CustomerID  int     `json:"customer_id" db:"customer_id"`
	CarID       int     `json:"car_id" db:"car_id"`
	TotalAmount float64 `json:"total_amount" db:"total_amount"`

	ScheduledAt time.Time  `json:"scheduled_at" db:"scheduled_at"`
	StartedAt   *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`

	Status  OrderStatus `json:"status" db:"status"`
	Comment *string     `json:"comment,omitempty" db:"comment"`

	Items []OrderItem `json:"items"`

	Car      *Car         `json:"car,omitempty"`
	Customer *Customer    `json:"customer,omitempty"`
	Media    []OrderMedia `json:"media,omitempty"`
}

type OrderItem struct {
	ID        int       `json:"id" db:"id"`
	OrderID   int       `json:"order_id" db:"order_id"`
	ServiceID int       `json:"service_id" db:"service_id"`
	Quantity  int       `json:"quantity" db:"quantity"`
	Price     float64   `json:"price" db:"price"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	ServiceName string  `json:"service_name" db:"service_name"`
	Comment     *string `json:"comment,omitempty" db:"comment"`
}

type OrderMedia struct {
	ID        int       `json:"id" db:"id"`
	OrderID   int       `json:"order_id" db:"order_id"`
	FileSize  *int64    `json:"file_size,omitempty" db:"file_size"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	Type       string  `json:"type" db:"type"`
	Stage      string  `json:"stage" db:"stage"`
	StorageKey string  `json:"storage_key" db:"storage_key"`
	MimeType   string  `json:"mime_type" db:"mime_type"`
	Comment    *string `json:"comment,omitempty" db:"comment"`

	// URL постоянная публичная ссылка на файл, вычисляется на лету из StorageKey
	// текущим драйвером хранилища и не хранится в БД.
	URL string `json:"url"`
}

// CreateOrderRequest представляет запрос на создание заказа
type CreateOrderRequest struct {
	CustomerID  int                      `json:"customer_id"`
	CarID       int                      `json:"car_id"`
	ScheduledAt time.Time                `json:"scheduled_at"`
	Status      string                   `json:"status"`
	Comment     *string                  `json:"comment"`
	Items       []CreateOrderItemRequest `json:"items"`
}

// CreateOrderItemRequest представляет запрос на создание услуги в заказе.
// Указывается либо ServiceID (используется существующая услуга), либо ServiceName
// (создается новая услуга в таблице services, а затем order_item с ее id).
type CreateOrderItemRequest struct {
	ServiceID int     `json:"service_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`

	ServiceName        string  `json:"service_name"`
	ServiceDescription *string `json:"service_description,omitempty"`
	ServiceIsActive    bool    `json:"service_is_active,omitempty"`
	Comment            *string `json:"comment"`
}
