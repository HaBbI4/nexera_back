package repository

import (
	"database/sql"
	"fmt"

	"nexera_back/internal/database"
	"nexera_back/internal/models"
)

// UserRepository описывает доступ к данным пользователей админ-панели.
// Сервисы зависят от этого интерфейса, а не от конкретной БД — в unit-тестах
// вместо Postgres подставляется фейковая реализация, реализующая тот же интерфейс.
type UserRepository interface {
	GetByEmail(email string) (*models.User, error)
	GetByID(id int) (*models.User, error)
	UpdateLastLogin(id int) error
}

// userRepository реализует UserRepository поверх PostgreSQL.
type userRepository struct {
	db *database.DB
}

// NewUserRepository создает репозиторий пользователей админ-панели на PostgreSQL.
func NewUserRepository(db *database.DB) UserRepository {
	return &userRepository{db: db}
}

// GetByEmail возвращает пользователя вместе с его ролью по email.
// Пользователь с deleted_at IS NOT NULL считается не найденным.
func (r *userRepository) GetByEmail(email string) (*models.User, error) {
	return r.scanUser(`
		SELECT u.id, u.name, u.email, u.password_hash, u.role_id, u.is_active,
		       u.last_login_at, u.created_at, u.updated_at,
		       r.id, r.name, r.code
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.email = $1 AND u.deleted_at IS NULL
	`, email)
}

// GetByID возвращает пользователя вместе с его ролью по ID.
func (r *userRepository) GetByID(id int) (*models.User, error) {
	return r.scanUser(`
		SELECT u.id, u.name, u.email, u.password_hash, u.role_id, u.is_active,
		       u.last_login_at, u.created_at, u.updated_at,
		       r.id, r.name, r.code
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.id = $1 AND u.deleted_at IS NULL
	`, id)
}

func (r *userRepository) scanUser(query string, arg interface{}) (*models.User, error) {
	user := &models.User{Role: &models.Role{}}

	err := r.db.QueryRow(query, arg).Scan(
		&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.RoleID, &user.IsActive,
		&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
		&user.Role.ID, &user.Role.Name, &user.Role.Code,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("error getting user: %w", err)
	}

	return user, nil
}

// UpdateLastLogin обновляет отметку времени последнего входа пользователя.
func (r *userRepository) UpdateLastLogin(id int) error {
	_, err := r.db.Exec(`UPDATE users SET last_login_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("error updating last login: %w", err)
	}
	return nil
}
