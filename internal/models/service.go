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

// UpdateServiceRequest представляет PATCH-запрос на обновление услуги: nil-поле значит
// "не менять", присутствующее — новое значение. Так через это поле нельзя явно сбросить
// description/price обратно в NULL ("цена по запросу"), только задать новое значение —
// то же осознанное упрощение, что и в UpdateBannerRequest.
type UpdateServiceRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Price       *int    `json:"price"`
	Position    *int    `json:"position"`
	IsPublished *bool   `json:"is_published"`
}
