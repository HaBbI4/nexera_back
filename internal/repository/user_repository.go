package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"nexera_back/internal/database"
	"nexera_back/internal/models"
)

// ErrUserNotFound возвращается, когда пользователя с указанным ID/email не существует
// или он мягко удален (deleted_at IS NOT NULL).
// ErrUserEmailTaken возвращается, когда email уже занят другим пользователем (users.email
// UNIQUE). Внимание: уникальность на колонке не частичная, поэтому email мягко удаленного
// пользователя тоже считается занятым.
var (
	ErrUserNotFound   = errors.New("user not found")
	ErrUserEmailTaken = errors.New("user email already taken")
)

// UserSortField перечисляет поля, по которым разрешена сортировка списка пользователей.
// Белый список по той же причине, что BannerSortField: колонку в ORDER BY нельзя
// параметризовать через $N, в SQL попадает только то, что есть в userSortColumns.
type UserSortField string

const (
	UserSortCreatedAt UserSortField = "created_at"
	UserSortName      UserSortField = "name"
	UserSortEmail     UserSortField = "email"
)

// userSortColumns отображает UserSortField в реальную колонку таблицы users.
var userSortColumns = map[UserSortField]string{
	UserSortCreatedAt: "u.created_at",
	UserSortName:      "u.name",
	UserSortEmail:     "u.email",
}

// UserFilter описывает фильтрацию, сортировку и пагинацию списка пользователей на уровне БД.
// Мягко удаленные пользователи в список не попадают никогда.
type UserFilter struct {
	// RoleID — вернуть только пользователей с этой ролью; nil — с любой.
	RoleID *int
	// IsActive — вернуть только активных (true) или только отключенных (false); nil — всех.
	IsActive *bool

	Sort  UserSortField
	Order SortOrder

	Limit  int
	Offset int
}

// UserRepository описывает доступ к данным пользователей админ-панели.
// Сервисы зависят от этого интерфейса, а не от конкретной БД — в unit-тестах
// вместо Postgres подставляется фейковая реализация, реализующая тот же интерфейс.
type UserRepository interface {
	List(filter UserFilter) ([]*models.User, error)
	GetByEmail(email string) (*models.User, error)
	GetByID(id int) (*models.User, error)
	// Create сохраняет нового пользователя, заполняя user.ID/CreatedAt/UpdatedAt значениями
	// из БД. user.Role не заполняется — роль подгружает сервис (см. UserService.Create).
	Create(user *models.User) error
	// Update полностью перезаписывает пользователя user.ID новым состоянием (включая
	// password_hash). Какие поля патчить, а какие оставить как есть — решает сервис
	// (см. UserService.Update).
	Update(user *models.User) error
	// Delete мягко удаляет пользователя: проставляет deleted_at, строка остается в БД.
	Delete(id int) error
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

// userSelect — общая часть SELECT для чтения пользователя вместе с его ролью;
// порядок колонок соответствует scanUserRow.
const userSelect = `
	SELECT u.id, u.name, u.email, u.password_hash, u.role_id, u.is_active,
	       u.last_login_at, u.created_at, u.updated_at,
	       r.id, r.name, r.code
	FROM users u
	JOIN roles r ON r.id = u.role_id
`

// rowScanner — общий интерфейс *sql.Row и *sql.Rows, чтобы один scanUserRow
// обслуживал и одиночные запросы, и списки.
type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanUserRow(row rowScanner) (*models.User, error) {
	user := &models.User{Role: &models.Role{}}

	err := row.Scan(
		&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.RoleID, &user.IsActive,
		&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
		&user.Role.ID, &user.Role.Name, &user.Role.Code,
	)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// List возвращает пользователей (вместе с ролями) по фильтру, с сортировкой и пагинацией.
func (r *userRepository) List(filter UserFilter) ([]*models.User, error) {
	query := userSelect + ` WHERE u.deleted_at IS NULL`
	args := []interface{}{}
	argIndex := 1

	if filter.RoleID != nil {
		query += fmt.Sprintf(" AND u.role_id = $%d", argIndex)
		args = append(args, *filter.RoleID)
		argIndex++
	}

	if filter.IsActive != nil {
		query += fmt.Sprintf(" AND u.is_active = $%d", argIndex)
		args = append(args, *filter.IsActive)
		argIndex++
	}

	column := userSortColumns[filter.Sort]
	if column == "" {
		column = userSortColumns[UserSortCreatedAt]
	}
	direction := "ASC"
	if filter.Order == SortDesc {
		direction = "DESC"
	}
	// id тем же направлением — чтобы строки с одинаковым значением сортируемого поля
	// не "плавали" между запросами страниц пагинации.
	query += fmt.Sprintf(" ORDER BY %s %s, u.id %s", column, direction, direction)

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
		return nil, fmt.Errorf("error getting users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user, err := scanUserRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

// GetByEmail возвращает пользователя вместе с его ролью по email.
// Пользователь с deleted_at IS NOT NULL считается не найденным.
func (r *userRepository) GetByEmail(email string) (*models.User, error) {
	return r.getOne(userSelect+` WHERE u.email = $1 AND u.deleted_at IS NULL`, email)
}

// GetByID возвращает пользователя вместе с его ролью по ID.
// Пользователь с deleted_at IS NOT NULL считается не найденным.
func (r *userRepository) GetByID(id int) (*models.User, error) {
	return r.getOne(userSelect+` WHERE u.id = $1 AND u.deleted_at IS NULL`, id)
}

func (r *userRepository) getOne(query string, arg interface{}) (*models.User, error) {
	user, err := scanUserRow(r.db.QueryRow(query, arg))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("error getting user: %w", err)
	}

	return user, nil
}

// Create сохраняет нового пользователя.
func (r *userRepository) Create(user *models.User) error {
	query := `
		INSERT INTO users (name, email, password_hash, role_id, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(query, user.Name, user.Email, user.PasswordHash, user.RoleID, user.IsActive).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrUserEmailTaken
		}
		if isForeignKeyViolation(err) {
			return ErrRoleNotFound
		}
		return fmt.Errorf("error creating user: %w", err)
	}

	return nil
}

// Update перезаписывает пользователя user.ID целиком новым состоянием и обновляет updated_at.
// Мягко удаленного пользователя обновить нельзя — для него вернется ErrUserNotFound.
func (r *userRepository) Update(user *models.User) error {
	query := `
		UPDATE users
		SET name = $1, email = $2, password_hash = $3, role_id = $4, is_active = $5, updated_at = NOW()
		WHERE id = $6 AND deleted_at IS NULL
		RETURNING updated_at
	`

	err := r.db.QueryRow(query, user.Name, user.Email, user.PasswordHash, user.RoleID, user.IsActive, user.ID).
		Scan(&user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		if isUniqueViolation(err) {
			return ErrUserEmailTaken
		}
		if isForeignKeyViolation(err) {
			return ErrRoleNotFound
		}
		return fmt.Errorf("error updating user: %w", err)
	}

	return nil
}

// Delete мягко удаляет пользователя по ID: проставляет deleted_at. Повторное удаление
// уже удаленного пользователя — ErrUserNotFound.
func (r *userRepository) Delete(id int) error {
	var deletedID int
	err := r.db.QueryRow(
		`UPDATE users SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL RETURNING id`,
		id,
	).Scan(&deletedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("error deleting user: %w", err)
	}

	return nil
}

// UpdateLastLogin обновляет отметку времени последнего входа пользователя.
func (r *userRepository) UpdateLastLogin(id int) error {
	_, err := r.db.Exec(`UPDATE users SET last_login_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("error updating last login: %w", err)
	}
	return nil
}
