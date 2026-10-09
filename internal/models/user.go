package models

import "time"

// User представляет пользователя админ-панели (сотрудника компании с доступом к управлению сайтом).
type User struct {
	ID          int        `json:"id" db:"id"`
	RoleID      int        `json:"role_id" db:"role_id"`
	IsActive    bool       `json:"is_active" db:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	Name         string `json:"name" db:"name"`
	Email        string `json:"email" db:"email"`
	PasswordHash string `json:"-" db:"password_hash"`

	Role *Role `json:"role,omitempty"`
}

// CreateUserRequest представляет запрос на создание пользователя админ-панели
type CreateUserRequest struct {
	RoleID   int    `json:"role_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
