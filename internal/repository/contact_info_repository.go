package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"nexera_back/internal/database"
	"nexera_back/internal/models"
)

// ErrContactInfoNotFound возвращается, когда записи contact_info с указанным ID не существует.
var ErrContactInfoNotFound = errors.New("contact info not found")

// ContactInfoSortField перечисляет поля, по которым разрешена сортировка списка contact_info.
type ContactInfoSortField string

const (
	ContactInfoSortPosition  ContactInfoSortField = "position"
	ContactInfoSortCreatedAt ContactInfoSortField = "created_at"
)

var contactInfoSortColumns = map[ContactInfoSortField]string{
	ContactInfoSortPosition:  "position",
	ContactInfoSortCreatedAt: "created_at",
}

// ContactInfoFilter описывает фильтрацию, сортировку и пагинацию списка contact_info на уровне БД.
type ContactInfoFilter struct {
	IsPublished *bool

	Sort  ContactInfoSortField
	Order SortOrder

	Limit  int
	Offset int
}

// ContactInfoRepository описывает доступ к данным contact_info.
type ContactInfoRepository interface {
	List(filter ContactInfoFilter) ([]*models.ContactInfo, error)
	GetByID(id int) (*models.ContactInfo, error)
	Create(info *models.ContactInfo) error
	Update(info *models.ContactInfo) error
	Delete(id int) error
}

type contactInfoRepository struct {
	db *database.DB
}

// NewContactInfoRepository создает репозиторий contact_info на PostgreSQL.
func NewContactInfoRepository(db *database.DB) ContactInfoRepository {
	return &contactInfoRepository{db: db}
}

func (r *contactInfoRepository) List(filter ContactInfoFilter) ([]*models.ContactInfo, error) {
	query := `
		SELECT id, type, value, label, position, is_published, created_at, updated_at
		FROM contact_info
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if filter.IsPublished != nil {
		query += fmt.Sprintf(" AND is_published = $%d", argIndex)
		args = append(args, *filter.IsPublished)
		argIndex++
	}

	column := contactInfoSortColumns[filter.Sort]
	if column == "" {
		column = contactInfoSortColumns[ContactInfoSortPosition]
	}
	direction := "ASC"
	if filter.Order == SortDesc {
		direction = "DESC"
	}
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
		return nil, fmt.Errorf("error getting contact info: %w", err)
	}
	defer rows.Close()

	var items []*models.ContactInfo
	for rows.Next() {
		info := &models.ContactInfo{}
		if err := rows.Scan(
			&info.ID, &info.Type, &info.Value, &info.Label, &info.Position, &info.IsPublished,
			&info.CreatedAt, &info.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan contact info: %w", err)
		}
		items = append(items, info)
	}

	return items, nil
}

func (r *contactInfoRepository) GetByID(id int) (*models.ContactInfo, error) {
	info := &models.ContactInfo{}

	query := `
		SELECT id, type, value, label, position, is_published, created_at, updated_at
		FROM contact_info
		WHERE id = $1
	`

	err := r.db.QueryRow(query, id).Scan(
		&info.ID, &info.Type, &info.Value, &info.Label, &info.Position, &info.IsPublished,
		&info.CreatedAt, &info.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrContactInfoNotFound
		}
		return nil, fmt.Errorf("error getting contact info: %w", err)
	}

	return info, nil
}

func (r *contactInfoRepository) Create(info *models.ContactInfo) error {
	query := `
		INSERT INTO contact_info (type, value, label, position, is_published)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(query, info.Type, info.Value, info.Label, info.Position, info.IsPublished).
		Scan(&info.ID, &info.CreatedAt, &info.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error creating contact info: %w", err)
	}

	return nil
}

func (r *contactInfoRepository) Update(info *models.ContactInfo) error {
	query := `
		UPDATE contact_info
		SET type = $1, value = $2, label = $3, position = $4, is_published = $5, updated_at = NOW()
		WHERE id = $6
		RETURNING updated_at
	`

	err := r.db.QueryRow(query, info.Type, info.Value, info.Label, info.Position, info.IsPublished, info.ID).
		Scan(&info.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrContactInfoNotFound
		}
		return fmt.Errorf("error updating contact info: %w", err)
	}

	return nil
}

func (r *contactInfoRepository) Delete(id int) error {
	var deletedID int
	err := r.db.QueryRow(`DELETE FROM contact_info WHERE id = $1 RETURNING id`, id).Scan(&deletedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrContactInfoNotFound
		}
		return fmt.Errorf("error deleting contact info: %w", err)
	}

	return nil
}
