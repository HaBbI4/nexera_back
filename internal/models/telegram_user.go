package models

import "time"

// TelegramUser представляет пользователя telegram-бота (владелец, сотрудник),
// которому разрешён доступ к боту и который получает уведомления о заявках и отзывах.
type TelegramUser struct {
	ID        int       `json:"id" db:"id"`
	ChatID    int64     `json:"chat_id" db:"chat_id"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	Name string `json:"name" db:"name"`
}

// CreateTelegramUserRequest представляет запрос на добавление пользователя бота.
// IsActive — указатель, чтобы отличить "не прислали" (nil — по умолчанию активен,
// как DEFAULT TRUE в telegram_users) от явного false.
type CreateTelegramUserRequest struct {
	ChatID   int64  `json:"chat_id"`
	Name     string `json:"name"`
	IsActive *bool  `json:"is_active"`
}

// UpdateTelegramUserRequest представляет PATCH-запрос на обновление пользователя бота:
// nil-поле значит "не менять", присутствующее — новое значение.
type UpdateTelegramUserRequest struct {
	ChatID   *int64  `json:"chat_id"`
	Name     *string `json:"name"`
	IsActive *bool   `json:"is_active"`
}
