package models

import "time"

// Service представляет услугу компании, отображаемую на сайте.
type Service struct {
	ID          int       `json:"id" db:"id"`
	Position    int       `json:"position" db:"position"`
	IsPublished bool      `json:"is_published" db:"is_published"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`

	Name        string  `json:"name" db:"name"`
	Description *string `json:"description,omitempty" db:"description"`
	Price       *int    `json:"price,omitempty" db:"price"`
}

// CreateServiceRequest представляет запрос на создание услуги
type CreateServiceRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Price       *int    `json:"price"`
	Position    int     `json:"position"`
	IsPublished bool    `json:"is_published"`
}
