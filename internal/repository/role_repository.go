package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"nexera_back/internal/database"
	"nexera_back/internal/models"
)

// ErrRoleNotFound возвращается, когда роли с указанным ID не существует — в том числе,
// когда role_id пользователя ссылается на несуществующую роль (нарушение внешнего ключа
// users.role_id -> roles.id).
// ErrRoleCodeTaken возвращается, когда code роли уже занят другой ролью (roles.code UNIQUE).
// ErrRoleInUse возвращается при удалении роли, на которую ссылаются пользователи
// (users.role_id -> roles(id) ON DELETE RESTRICT).
var (
	ErrRoleNotFound  = errors.New("role not found")
	ErrRoleCodeTaken = errors.New("role code already taken")
	ErrRoleInUse     = errors.New("role is assigned to users")
)

// RoleRepository описывает доступ к данным ролей пользователей админ-панели.
type RoleRepository interface {
	List() ([]*models.Role, error)
	GetByID(id int) (*models.Role, error)
	// Create сохраняет новую роль, заполняя role.ID/CreatedAt/UpdatedAt значениями из БД.
	Create(role *models.Role) error
	// Update полностью перезаписывает роль role.ID новым состоянием. Какие поля патчить,
	// а какие оставить как есть — решает сервис (см. RoleService.Update).
	Update(role *models.Role) error
	// Delete удаляет роль. Роль, назначенную хотя бы одному пользователю (включая
	// мягко удаленных — строка в users остается), удалить нельзя: ErrRoleInUse.
	Delete(id int) error
}

// roleRepository реализует RoleRepository поверх PostgreSQL.
type roleRepository struct {
	db *database.DB
}

// NewRoleRepository создает репозиторий ролей на PostgreSQL.
func NewRoleRepository(db *database.DB) RoleRepository {
	return &roleRepository{db: db}
}

// List возвращает все роли, отсортированные по ID. Ролей в системе единицы (справочник,
// а не пользовательские данные), поэтому ни пагинация, ни фильтрация здесь не нужны.
func (r *roleRepository) List() ([]*models.Role, error) {
	rows, err := r.db.Query(`SELECT id, name, code, created_at, updated_at FROM roles ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("error getting roles: %w", err)
	}
	defer rows.Close()

	var roles []*models.Role
	for rows.Next() {
		role := &models.Role{}
		if err := rows.Scan(&role.ID, &role.Name, &role.Code, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// GetByID возвращает роль по ID.
func (r *roleRepository) GetByID(id int) (*models.Role, error) {
	role := &models.Role{}

	err := r.db.QueryRow(`SELECT id, name, code, created_at, updated_at FROM roles WHERE id = $1`, id).
		Scan(&role.ID, &role.Name, &role.Code, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("error getting role: %w", err)
	}

	return role, nil
}

// Create сохраняет новую роль.
func (r *roleRepository) Create(role *models.Role) error {
	query := `INSERT INTO roles (name, code) VALUES ($1, $2) RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(query, role.Name, role.Code).Scan(&role.ID, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrRoleCodeTaken
		}
		return fmt.Errorf("error creating role: %w", err)
	}

	return nil
}

// Update перезаписывает роль role.ID целиком новым состоянием и обновляет updated_at.
func (r *roleRepository) Update(role *models.Role) error {
	query := `UPDATE roles SET name = $1, code = $2, updated_at = NOW() WHERE id = $3 RETURNING updated_at`

	err := r.db.QueryRow(query, role.Name, role.Code, role.ID).Scan(&role.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRoleNotFound
		}
		if isUniqueViolation(err) {
			return ErrRoleCodeTaken
		}
		return fmt.Errorf("error updating role: %w", err)
	}

	return nil
}

// Delete удаляет роль по ID. users.role_id -> roles(id) ON DELETE RESTRICT, поэтому
// роль, назначенная пользователям, не удаляется — Postgres отвечает 23503, который
// транслируется в ErrRoleInUse.
func (r *roleRepository) Delete(id int) error {
	var deletedID int
	err := r.db.QueryRow(`DELETE FROM roles WHERE id = $1 RETURNING id`, id).Scan(&deletedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRoleNotFound
		}
		if isForeignKeyViolation(err) {
			return ErrRoleInUse
		}
		return fmt.Errorf("error deleting role: %w", err)
	}

	return nil
}
