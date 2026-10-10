package models

import "time"

// Review представляет отзыв клиента. Публикуется на сайте только после модерации (IsApproved).
// Создание отзыва публикует событие в Kafka (topic "reviews") для уведомления в telegram-bot.
//
// Review содержит контакты клиента (Phone, Email) и отдается только админ-панели. Наружу,
// на сайт, отзыв уходит как PublicReview — см. ToPublic.
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

// PublicReview — отзыв в том виде, в котором его можно показать посетителю сайта: без
// контактов клиента и служебных полей модерации. Отдельный тип, а не обнуление полей
// Review в хендлере — у этой структуры просто нет полей, через которые могли бы утечь
// телефон или email, как бы ни менялся код публичных эндпоинтов.
type PublicReview struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Rating    int       `json:"rating"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// ToPublic возвращает публичное представление отзыва.
func (r *Review) ToPublic() *PublicReview {
	return &PublicReview{
		ID:        r.ID,
		Name:      r.Name,
		Rating:    r.Rating,
		Text:      r.Text,
		CreatedAt: r.CreatedAt,
	}
}

// ReviewSummary — сводка по отзывам для блока "4.8 из 5, 37 отзывов" на сайте.
// AverageRating округлен до одного знака; 0, если отзывов нет.
type ReviewSummary struct {
	Count         int     `json:"count"`
	AverageRating float64 `json:"average_rating"`
}

// CreateReviewRequest представляет запрос клиента на создание отзыва
type CreateReviewRequest struct {
	Name   string  `json:"name"`
	Phone  string  `json:"phone"`
	Email  *string `json:"email"`
	Rating int     `json:"rating"`
	Text   string  `json:"text"`
	// Website — honeypot: скрытое поле формы, человек его не видит и не заполняет.
	// Заполнено — запрос отправил бот (см. services.ErrSpamDetected). Должно быть пустым.
	Website string `json:"website"`
}
