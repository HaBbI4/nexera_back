package models

import "time"

// Review представляет отзыв клиента. Публикуется на сайте только после модерации (IsApproved).
// Создание отзыва публикует событие в Kafka (topic "reviews") для уведомления в telegram-bot.
type Review struct {
	ID         int        `json:"id" db:"id"`
	Rating     int        `json:"rating" db:"rating"`
	IsApproved bool       `json:"is_approved" db:"is_approved"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	Name  string  `json:"name" db:"name"`
	Phone string  `json:"phone" db:"phone"`
	Email *string `json:"email,omitempty" db:"email"`
	Text  string  `json:"text" db:"text"`
}

// CreateReviewRequest представляет запрос клиента на создание отзыва
type CreateReviewRequest struct {
	Name   string  `json:"name"`
	Phone  string  `json:"phone"`
	Email  *string `json:"email"`
	Rating int     `json:"rating"`
	Text   string  `json:"text"`
}
