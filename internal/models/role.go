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

// RoleCodeAdmin — код роли администратора: только ей доступно управление пользователями
// и ролями (см. маршруты /api/v1/admin/users и /api/v1/admin/roles в cmd/server/main.go).
const RoleCodeAdmin = "admin"

// CreateRoleRequest представляет запрос на создание роли
type CreateRoleRequest struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

// UpdateRoleRequest представляет PATCH-запрос на обновление роли: nil-поле значит
// "не менять", присутствующее — новое значение.
type UpdateRoleRequest struct {
	Name *string `json:"name"`
	Code *string `json:"code"`
}
