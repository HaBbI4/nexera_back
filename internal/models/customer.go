package models

import "time"

// Customer Представляет пользователя в системе
type Customer struct {
	ID        int       `json:"id" db:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	Name    string `json:"name" db:"name"`
	Phone   string `json:"phone" db:"phone"`
	Email   string `json:"email" db:"email"`
	Comment string `json:"comment" db:"comment"`
}

// CreateCustomerRequest представляет запрос на создание клиента
type CreateCustomerRequest struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Comment string `json:"comment"`
}
