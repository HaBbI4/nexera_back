package models

import "time"

// TelegramUser представляет пользователя telegram-бота (владелец, сотрудник),
// которому разрешён доступ к боту и который получает уведомления о заказах.
type TelegramUser struct {
	ID        int       `json:"id" db:"id"`
	ChatID    int64     `json:"chat_id" db:"chat_id"`
	Name      string    `json:"name" db:"name"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
