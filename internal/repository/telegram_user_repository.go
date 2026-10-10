package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"nexera_back/internal/database"
	"nexera_back/internal/models"
)

var (
	// ErrTelegramUserNotFound возвращается, когда пользователя бота с указанным ID (chat_id)
	// не существует.
	ErrTelegramUserNotFound = errors.New("telegram user not found")
	// ErrTelegramUserChatIDTaken возвращается, когда chat_id уже привязан к другому
	// пользователю бота (нарушение уникального индекса telegram_users_chat_id_uq).
	ErrTelegramUserChatIDTaken = errors.New("telegram user chat_id already taken")
)

// TelegramUserSortField перечисляет поля, по которым разрешена сортировка списка
// пользователей бота. Белый список по той же причине, что BannerSortField.
type TelegramUserSortField string

const (
	TelegramUserSortCreatedAt TelegramUserSortField = "created_at"
	TelegramUserSortName      TelegramUserSortField = "name"
)

// telegramUserSortColumns отображает TelegramUserSortField в реальную колонку таблицы.
var telegramUserSortColumns = map[TelegramUserSortField]string{
	TelegramUserSortCreatedAt: "created_at",
	TelegramUserSortName:      "name",
}

// TelegramUserFilter описывает фильтрацию, сортировку и пагинацию списка пользователей бота.
type TelegramUserFilter struct {
	// IsActive — только активные (true) или только отключенные (false); nil — все.
	IsActive *bool

	Sort  TelegramUserSortField
	Order SortOrder

	Limit  int
	Offset int
}

// TelegramUserRepository описывает доступ к данным пользователей telegram-бота.
type TelegramUserRepository interface {
	List(filter TelegramUserFilter) ([]*models.TelegramUser, error)
	GetByID(id int) (*models.TelegramUser, error)
	// GetByChatID ищет пользователя по Telegram chat_id — для авторизации в боте.
	GetByChatID(chatID int64) (*models.TelegramUser, error)
	// Create сохраняет нового пользователя, заполняя ID/CreatedAt/UpdatedAt значениями из БД.
	Create(user *models.TelegramUser) error
	// Update полностью перезаписывает пользователя user.ID новым состоянием. Какие поля
	// патчить — решает сервис (см. TelegramUserService.Update).
	Update(user *models.TelegramUser) error
	Delete(id int) error
}

// telegramUserRepository реализует TelegramUserRepository поверх PostgreSQL.
type telegramUserRepository struct {
	db *database.DB
}

// NewTelegramUserRepository создает репозиторий пользователей бота на PostgreSQL.
func NewTelegramUserRepository(db *database.DB) TelegramUserRepository {
	return &telegramUserRepository{db: db}
}

const telegramUserColumns = `id, chat_id, name, is_active, created_at, updated_at`

func scanTelegramUserRow(row rowScanner) (*models.TelegramUser, error) {
	user := &models.TelegramUser{}
	err := row.Scan(&user.ID, &user.ChatID, &user.Name, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

// List возвращает пользователей бота по фильтру, с сортировкой и пагинацией.
func (r *telegramUserRepository) List(filter TelegramUserFilter) ([]*models.TelegramUser, error) {
	query := `SELECT ` + telegramUserColumns + ` FROM telegram_users WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if filter.IsActive != nil {
		query += fmt.Sprintf(" AND is_active = $%d", argIndex)
		args = append(args, *filter.IsActive)
		argIndex++
	}

	column := telegramUserSortColumns[filter.Sort]
	if column == "" {
		column = telegramUserSortColumns[TelegramUserSortCreatedAt]
	}
	direction := "ASC"
	if filter.Order == SortDesc {
		direction = "DESC"
	}
	// id тем же направлением — чтобы строки с одинаковым значением сортируемого поля
	// не "плавали" между запросами страниц пагинации.
	query += fmt.Sprintf(" ORDER BY %s %s, id %s", column, direction, direction)

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error getting telegram users: %w", err)
	}
	defer rows.Close()

	var users []*models.TelegramUser
	for rows.Next() {
		user, err := scanTelegramUserRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan telegram user: %w", err)
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// GetByID возвращает пользователя бота по ID вне зависимости от is_active.
func (r *telegramUserRepository) GetByID(id int) (*models.TelegramUser, error) {
	query := `SELECT ` + telegramUserColumns + ` FROM telegram_users WHERE id = $1`

	user, err := scanTelegramUserRow(r.db.QueryRow(query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTelegramUserNotFound
		}
		return nil, fmt.Errorf("error getting telegram user: %w", err)
	}

	return user, nil
}

// GetByChatID возвращает пользователя бота по chat_id вне зависимости от is_active —
// пускать ли его в бот, решает вызывающий код.
func (r *telegramUserRepository) GetByChatID(chatID int64) (*models.TelegramUser, error) {
	query := `SELECT ` + telegramUserColumns + ` FROM telegram_users WHERE chat_id = $1`

	user, err := scanTelegramUserRow(r.db.QueryRow(query, chatID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTelegramUserNotFound
		}
		return nil, fmt.Errorf("error getting telegram user by chat_id: %w", err)
	}

	return user, nil
}

// Create сохраняет нового пользователя бота.
func (r *telegramUserRepository) Create(user *models.TelegramUser) error {
	query := `
		INSERT INTO telegram_users (chat_id, name, is_active)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(query, user.ChatID, user.Name, user.IsActive).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrTelegramUserChatIDTaken
		}
		return fmt.Errorf("error creating telegram user: %w", err)
	}

	return nil
}

// Update перезаписывает пользователя бота user.ID целиком новым состоянием и обновляет updated_at.
func (r *telegramUserRepository) Update(user *models.TelegramUser) error {
	query := `
		UPDATE telegram_users
		SET chat_id = $1, name = $2, is_active = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING updated_at
	`

	err := r.db.QueryRow(query, user.ChatID, user.Name, user.IsActive, user.ID).Scan(&user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTelegramUserNotFound
		}
		if isUniqueViolation(err) {
			return ErrTelegramUserChatIDTaken
		}
		return fmt.Errorf("error updating telegram user: %w", err)
	}

	return nil
}

// Delete удаляет пользователя бота по ID — он сразу теряет доступ к боту и перестает
// получать уведомления.
func (r *telegramUserRepository) Delete(id int) error {
	var deletedID int
	err := r.db.QueryRow(`DELETE FROM telegram_users WHERE id = $1 RETURNING id`, id).Scan(&deletedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTelegramUserNotFound
		}
		return fmt.Errorf("error deleting telegram user: %w", err)
	}

	return nil
}
