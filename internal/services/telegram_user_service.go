package services

import (
	"database/sql"
	"fmt"
	"nexera_back/internal/database"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
)

// TelegramUserService представляет сервис для работы с пользователями telegram-бота
type TelegramUserService struct {
	db  *database.DB
	log *logger.Logger
}

// NewTelegramUserService создает новый сервис для работы с пользователями telegram-бота
func NewTelegramUserService(db *database.DB, log *logger.Logger) *TelegramUserService {
	return &TelegramUserService{
		db:  db,
		log: log,
	}
}

// GetActiveUsers возвращает всех активных пользователей бота (получателей уведомлений)
func (svc *TelegramUserService) GetActiveUsers() ([]*models.TelegramUser, error) {
	query := `
		SELECT id, chat_id, name, is_active, created_at, updated_at
		FROM telegram_users
		WHERE is_active = TRUE
		ORDER BY id
	`

	rows, err := svc.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error getting active telegram users: %w", err)
	}
	defer rows.Close()

	var users []*models.TelegramUser
	for rows.Next() {
		user := &models.TelegramUser{}
		if err := rows.Scan(
			&user.ID, &user.ChatID, &user.Name, &user.IsActive,
			&user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error scan telegram user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

// GetByChatID возвращает пользователя бота по chat_id. Используется middleware авторизации:
// доступ к боту разрешен, только если найдена активная запись.
func (svc *TelegramUserService) GetByChatID(chatID int64) (*models.TelegramUser, error) {
	user := &models.TelegramUser{}

	query := `
		SELECT id, chat_id, name, is_active, created_at, updated_at
		FROM telegram_users
		WHERE chat_id = $1
	`

	err := svc.db.QueryRow(query, chatID).Scan(
		&user.ID, &user.ChatID, &user.Name, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("telegram user not found")
		}
		return nil, fmt.Errorf("error getting telegram user: %w", err)
	}

	return user, nil
}
