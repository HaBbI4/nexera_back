package models

import "time"

// Role представляет роль пользователя админ-панели (например: admin, manager) —
// используется для разграничения прав доступа.
type Role struct {
	ID        int       `json:"id" db:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	Name string `json:"name" db:"name"`
	Code string `json:"code" db:"code"`
}
