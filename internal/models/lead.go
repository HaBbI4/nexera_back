package models

import "time"

// Lead представляет заявку клиента с сайта (контактная форма и т.п.).
// Создание заявки публикует событие в Kafka (topic "leads"), которое читает telegram-bot
// и присылает уведомление владельцу/сотрудникам.
type Lead struct {
	ID        int       `json:"id" db:"id"`
	ServiceID *int      `json:"service_id,omitempty" db:"service_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	Name    string  `json:"name" db:"name"`
	Phone   string  `json:"phone" db:"phone"`
	Email   *string `json:"email,omitempty" db:"email"`
	Message *string `json:"message,omitempty" db:"message"`
	Source  string  `json:"source" db:"source"`

	Service *Service `json:"service,omitempty"`
}

// CreateLeadRequest представляет запрос клиента на создание заявки
type CreateLeadRequest struct {
	ServiceID *int    `json:"service_id"`
	Name      string  `json:"name"`
	Phone     string  `json:"phone"`
	Email     *string `json:"email"`
	Message   *string `json:"message"`
	Source    string  `json:"source"`
	// Website — honeypot: скрытое поле формы, человек его не видит и не заполняет.
	// Заполнено — запрос отправил бот (см. services.ErrSpamDetected). Должно быть пустым.
	Website string `json:"website"`
}
