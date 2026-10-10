package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"nexera_back/internal/database"
	"nexera_back/internal/models"
)

// ErrFAQNotFound возвращается, когда записи FAQ с указанным ID не существует.
var ErrFAQNotFound = errors.New("faq not found")

// FAQSortField перечисляет поля, по которым разрешена сортировка списка FAQ.
// Белый список по той же причине, что BannerSortField: колонку в ORDER BY нельзя
// параметризовать через $N, в SQL попадает только то, что есть в faqSortColumns.
type FAQSortField string

const (
	FAQSortPosition  FAQSortField = "position"
	FAQSortCreatedAt FAQSortField = "created_at"
)

// faqSortColumns отображает FAQSortField в реальную колонку таблицы faq.
var faqSortColumns = map[FAQSortField]string{
	FAQSortPosition:  "position",
	FAQSortCreatedAt: "created_at",
}

// FAQFilter описывает фильтрацию, сортировку и пагинацию списка FAQ на уровне БД.
type FAQFilter struct {
	// IsPublished — вернуть только опубликованные (true) или только скрытые (false) записи;
	// nil — все. Публичный путь всегда передает true — решение об этом принимает сервис
	// (бизнес-правило), а не вызывающий код.
	IsPublished *bool

	Sort  FAQSortField
	Order SortOrder

	Limit  int
	Offset int
}

// FAQRepository описывает доступ к данным FAQ.
type FAQRepository interface {
	List(filter FAQFilter) ([]*models.FAQ, error)
	GetByID(id int) (*models.FAQ, error)
	// Create сохраняет новую запись FAQ, заполняя faq.ID/CreatedAt/UpdatedAt значениями из БД.
	Create(faq *models.FAQ) error
	// Update полностью перезаписывает запись FAQ faq.ID новым состоянием. Какие поля патчить,
	// а какие оставить как есть — решает сервис (см. FAQService.Update).
	Update(faq *models.FAQ) error
	Delete(id int) error
}

// faqRepository реализует FAQRepository поверх PostgreSQL.
type faqRepository struct {
	db *database.DB
}

// NewFAQRepository создает репозиторий FAQ на PostgreSQL.
func NewFAQRepository(db *database.DB) FAQRepository {
	return &faqRepository{db: db}
}

// List возвращает записи FAQ по фильтру, с сортировкой и пагинацией.
func (r *faqRepository) List(filter FAQFilter) ([]*models.FAQ, error) {
	query := `
		SELECT id, question, answer, position, is_published, created_at, updated_at
		FROM faq
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if filter.IsPublished != nil {
		query += fmt.Sprintf(" AND is_published = $%d", argIndex)
		args = append(args, *filter.IsPublished)
		argIndex++
	}

	column := faqSortColumns[filter.Sort]
	if column == "" {
		column = faqSortColumns[FAQSortPosition]
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
		return nil, fmt.Errorf("error getting faq: %w", err)
	}
	defer rows.Close()

	var faqs []*models.FAQ
	for rows.Next() {
		faq := &models.FAQ{}
		if err := rows.Scan(
			&faq.ID, &faq.Question, &faq.Answer, &faq.Position, &faq.IsPublished,
			&faq.CreatedAt, &faq.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan faq: %w", err)
		}
		faqs = append(faqs, faq)
	}

	return faqs, nil
}

// GetByID возвращает запись FAQ по ID вне зависимости от того, опубликована ли она —
// решение, показывать ли ее, принимает сервис.
func (r *faqRepository) GetByID(id int) (*models.FAQ, error) {
	faq := &models.FAQ{}

	query := `
		SELECT id, question, answer, position, is_published, created_at, updated_at
		FROM faq
		WHERE id = $1
	`

	err := r.db.QueryRow(query, id).Scan(
		&faq.ID, &faq.Question, &faq.Answer, &faq.Position, &faq.IsPublished,
		&faq.CreatedAt, &faq.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrFAQNotFound
		}
		return nil, fmt.Errorf("error getting faq: %w", err)
	}

	return faq, nil
}

// Create сохраняет новую запись FAQ.
func (r *faqRepository) Create(faq *models.FAQ) error {
	query := `
		INSERT INTO faq (question, answer, position, is_published)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(query, faq.Question, faq.Answer, faq.Position, faq.IsPublished).
		Scan(&faq.ID, &faq.CreatedAt, &faq.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error creating faq: %w", err)
	}

	return nil
}

// Update перезаписывает запись FAQ faq.ID целиком новым состоянием и обновляет updated_at.
func (r *faqRepository) Update(faq *models.FAQ) error {
	query := `
		UPDATE faq
		SET question = $1, answer = $2, position = $3, is_published = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at
	`

	err := r.db.QueryRow(query, faq.Question, faq.Answer, faq.Position, faq.IsPublished, faq.ID).
		Scan(&faq.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrFAQNotFound
		}
		return fmt.Errorf("error updating faq: %w", err)
	}

	return nil
}

// Delete удаляет запись FAQ по ID.
func (r *faqRepository) Delete(id int) error {
	var deletedID int
	err := r.db.QueryRow(`DELETE FROM faq WHERE id = $1 RETURNING id`, id).Scan(&deletedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrFAQNotFound
		}
		return fmt.Errorf("error deleting faq: %w", err)
	}

	return nil
}
