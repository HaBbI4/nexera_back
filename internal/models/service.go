package models

import "time"

// Service представляет модель услуги
type Service struct {
	ID        int       `json:"id" db:"id"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	Name        string  `json:"name" db:"name"`
	Description *string `json:"description,omitempty" db:"description"`
}

// CreateServiceRequest представляет запрос на создание услуги
type CreateServiceRequest struct {
	IsActive    bool    `json:"is_active"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}
