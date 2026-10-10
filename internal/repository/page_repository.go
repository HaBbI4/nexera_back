package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"nexera_back/internal/database"
	"nexera_back/internal/models"
)

// ErrPageSlugTaken возвращается, когда slug страницы уже занят другой страницей
// (нарушение уникального индекса pages_slug_uq).
var ErrPageSlugTaken = errors.New("page slug already taken")

// PageRepository описывает доступ к данным страниц.
type PageRepository interface {
	List() ([]*models.Page, error)
	GetByID(id int) (*models.Page, error)
	// Create сохраняет новую страницу, заполняя page.ID/CreatedAt значениями из БД.
	Create(page *models.Page) error
	// Update полностью перезаписывает страницу page.ID новым состоянием. Какие поля патчить,
	// а какие оставить как есть — решает сервис (см. PageService.Update).
	Update(page *models.Page) error
	// Delete удаляет страницу. Внимание: banners.page_id ссылается на pages(id)
	// ON DELETE CASCADE — удаление страницы удалит и все ее баннеры.
	Delete(id int) error
}

// pageRepository реализует PageRepository поверх PostgreSQL.
type pageRepository struct {
	db *database.DB
}

// NewPageRepository создает репозиторий страниц на PostgreSQL.
func NewPageRepository(db *database.DB) PageRepository {
	return &pageRepository{db: db}
}

// List возвращает все страницы, отсортированные по ID. Страниц в системе мало (это
// справочник сайта-визитки, а не пользовательские данные), поэтому ни пагинация,
// ни фильтрация здесь не нужны.
func (r *pageRepository) List() ([]*models.Page, error) {
	rows, err := r.db.Query(`SELECT id, slug, name, created_at FROM pages ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("error getting pages: %w", err)
	}
	defer rows.Close()

	var pages []*models.Page
	for rows.Next() {
		page := &models.Page{}
		if err := rows.Scan(&page.ID, &page.Slug, &page.Name, &page.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan page: %w", err)
		}
		pages = append(pages, page)
	}

	return pages, nil
}

// GetByID возвращает страницу по ID (без баннеров — их подгружает сервис отдельно,
// см. PageService.GetPublicByIDWithBanners).
func (r *pageRepository) GetByID(id int) (*models.Page, error) {
	page := &models.Page{}

	err := r.db.QueryRow(`SELECT id, slug, name, created_at FROM pages WHERE id = $1`, id).
		Scan(&page.ID, &page.Slug, &page.Name, &page.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrPageNotFound
		}
		return nil, fmt.Errorf("error getting page: %w", err)
	}

	return page, nil
}

// Create сохраняет новую страницу.
func (r *pageRepository) Create(page *models.Page) error {
	query := `INSERT INTO pages (slug, name) VALUES ($1, $2) RETURNING id, created_at`

	err := r.db.QueryRow(query, page.Slug, page.Name).Scan(&page.ID, &page.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrPageSlugTaken
		}
		return fmt.Errorf("error creating page: %w", err)
	}

	return nil
}

// Update перезаписывает страницу page.ID целиком новым состоянием.
func (r *pageRepository) Update(page *models.Page) error {
	query := `UPDATE pages SET slug = $1, name = $2 WHERE id = $3 RETURNING id`

	err := r.db.QueryRow(query, page.Slug, page.Name, page.ID).Scan(&page.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPageNotFound
		}
		if isUniqueViolation(err) {
			return ErrPageSlugTaken
		}
		return fmt.Errorf("error updating page: %w", err)
	}

	return nil
}

// Delete удаляет страницу по ID. banners.page_id -> pages(id) ON DELETE CASCADE,
// поэтому вместе со страницей удалятся и все ее баннеры.
func (r *pageRepository) Delete(id int) error {
	var deletedID int
	err := r.db.QueryRow(`DELETE FROM pages WHERE id = $1 RETURNING id`, id).Scan(&deletedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPageNotFound
		}
		return fmt.Errorf("error deleting page: %w", err)
	}

	return nil
}

// isUniqueViolation сообщает, является ли err нарушением уникального индекса Postgres
// (код 23505) — тем же способом, что isForeignKeyViolation определяет код 23503.
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}
