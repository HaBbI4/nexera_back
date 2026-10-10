package models

import "time"

// FAQ представляет вопрос-ответ, отображаемый на страницах сайта.
type FAQ struct {
	ID          int       `json:"id" db:"id"`
	Position    int       `json:"position" db:"position"`
	IsPublished bool      `json:"is_published" db:"is_published"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`

	Question string `json:"question" db:"question"`
	Answer   string `json:"answer" db:"answer"`
}

// CreateFAQRequest представляет запрос на создание записи FAQ
type CreateFAQRequest struct {
	Question    string `json:"question"`
	Answer      string `json:"answer"`
	Position    int    `json:"position"`
	IsPublished bool   `json:"is_published"`
}

// UpdateFAQRequest представляет PATCH-запрос на обновление записи FAQ: nil-поле значит
// "не менять", присутствующее — новое значение.
type UpdateFAQRequest struct {
	Question    *string `json:"question"`
	Answer      *string `json:"answer"`
	Position    *int    `json:"position"`
	IsPublished *bool   `json:"is_published"`
}
