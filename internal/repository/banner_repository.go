package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"

	"nexera_back/internal/database"
	"nexera_back/internal/models"
)

// ErrBannerNotFound возвращается, когда баннера с указанным ID не существует.
// ErrPageNotFound возвращается, когда page_id баннера ссылается на несуществующую страницу
// (нарушение внешнего ключа banners.page_id -> pages.id).
var (
	ErrBannerNotFound = errors.New("banner not found")
	ErrPageNotFound   = errors.New("page not found")
)

// BannerSortField перечисляет поля, по которым разрешена сортировка списка баннеров.
// Это белый список — колонку в ORDER BY нельзя параметризовать через $N, поэтому
// в SQL попадает только то, что есть в bannerSortColumns, а не сырой ввод клиента.
type BannerSortField string

const (
	BannerSortPosition  BannerSortField = "position"
	BannerSortCreatedAt BannerSortField = "created_at"
)

// bannerSortColumns отображает BannerSortField в реальную колонку таблицы banners.
var bannerSortColumns = map[BannerSortField]string{
	BannerSortPosition:  "position",
	BannerSortCreatedAt: "created_at",
}

// BannerFilter описывает фильтрацию, сортировку и пагинацию списка баннеров на уровне БД.
type BannerFilter struct {
	// PageID — вернуть баннеры только одной страницы; nil — баннеры всех страниц.
	PageID *int
	// OnlyVisible — вернуть только баннеры, видимые прямо сейчас: is_active = true
	// и ActiveAt попадает в окно [starts_at, ends_at]. Решение о том, применять ли это
	// ограничение, принимает сервис (бизнес-правило), а не вызывающий код.
	OnlyVisible bool
	ActiveAt    time.Time

	Sort  BannerSortField
	Order SortOrder

	Limit  int
	Offset int
}

// BannerRepository описывает доступ к данным баннеров.
type BannerRepository interface {
	List(filter BannerFilter) ([]*models.Banner, error)
	GetByID(id int) (*models.Banner, error)
	// Create сохраняет новый баннер, заполняя banner.ID/CreatedAt/UpdatedAt значениями из БД.
	Create(banner *models.Banner) error
	// Update полностью перезаписывает баннер banner.ID новым состоянием. Какие поля патчить,
	// а какие оставить как есть — решает сервис (см. BannerService.Update), репозиторий
	// всегда получает уже готовую к сохранению сущность целиком.
	Update(banner *models.Banner) error
	Delete(id int) error
}

// bannerRepository реализует BannerRepository поверх PostgreSQL.
type bannerRepository struct {
	db *database.DB
}

// NewBannerRepository создает репозиторий баннеров на PostgreSQL.
func NewBannerRepository(db *database.DB) BannerRepository {
	return &bannerRepository{db: db}
}

// List возвращает баннеры по фильтру, с сортировкой и пагинацией.
func (r *bannerRepository) List(filter BannerFilter) ([]*models.Banner, error) {
	query := `
		SELECT id, page_id, title, subtitle, description, link_title, link_url, image_url,
		       position, starts_at, ends_at, is_active, created_at, updated_at
		FROM banners
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if filter.PageID != nil {
		query += fmt.Sprintf(" AND page_id = $%d", argIndex)
		args = append(args, *filter.PageID)
		argIndex++
	}

	if filter.OnlyVisible {
		query += fmt.Sprintf(" AND is_active = true AND (starts_at IS NULL OR starts_at <= $%d) AND (ends_at IS NULL OR ends_at >= $%d)", argIndex, argIndex)
		args = append(args, filter.ActiveAt)
		argIndex++
	}

	column := bannerSortColumns[filter.Sort]
	if column == "" {
		column = bannerSortColumns[BannerSortPosition]
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
		return nil, fmt.Errorf("error getting banners: %w", err)
	}
	defer rows.Close()

	var banners []*models.Banner
	for rows.Next() {
		banner := &models.Banner{}
		if err := rows.Scan(
			&banner.ID, &banner.PageID, &banner.Title, &banner.SubTitle, &banner.Description,
			&banner.LinkTitle, &banner.LinkURL, &banner.ImageURL,
			&banner.Position, &banner.StartsAt, &banner.EndsAt, &banner.IsActive,
			&banner.CreatedAt, &banner.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan banner: %w", err)
		}
		banners = append(banners, banner)
	}

	return banners, nil
}

// GetByID возвращает баннер по ID вне зависимости от его видимости —
// решение, показывать ли его, принимает сервис.
func (r *bannerRepository) GetByID(id int) (*models.Banner, error) {
	banner := &models.Banner{}

	query := `
		SELECT id, page_id, title, subtitle, description, link_title, link_url, image_url,
		       position, starts_at, ends_at, is_active, created_at, updated_at
		FROM banners
		WHERE id = $1
	`

	err := r.db.QueryRow(query, id).Scan(
		&banner.ID, &banner.PageID, &banner.Title, &banner.SubTitle, &banner.Description,
		&banner.LinkTitle, &banner.LinkURL, &banner.ImageURL,
		&banner.Position, &banner.StartsAt, &banner.EndsAt, &banner.IsActive,
		&banner.CreatedAt, &banner.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrBannerNotFound
		}
		return nil, fmt.Errorf("error getting banner: %w", err)
	}

	return banner, nil
}

// Create сохраняет новый баннер.
func (r *bannerRepository) Create(banner *models.Banner) error {
	query := `
		INSERT INTO banners (page_id, title, subtitle, description, link_title, link_url,
		                      image_url, position, starts_at, ends_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		banner.PageID, banner.Title, banner.SubTitle, banner.Description, banner.LinkTitle, banner.LinkURL,
		banner.ImageURL, banner.Position, banner.StartsAt, banner.EndsAt, banner.IsActive,
	).Scan(&banner.ID, &banner.CreatedAt, &banner.UpdatedAt)
	if err != nil {
		if isForeignKeyViolation(err) {
			return ErrPageNotFound
		}
		return fmt.Errorf("error creating banner: %w", err)
	}

	return nil
}

// Update перезаписывает баннер banner.ID целиком новым состоянием и обновляет updated_at.
func (r *bannerRepository) Update(banner *models.Banner) error {
	query := `
		UPDATE banners
		SET page_id = $1, title = $2, subtitle = $3, description = $4, link_title = $5,
		    link_url = $6, image_url = $7, position = $8, starts_at = $9, ends_at = $10,
		    is_active = $11, updated_at = NOW()
		WHERE id = $12
		RETURNING updated_at
	`

	err := r.db.QueryRow(
		query,
		banner.PageID, banner.Title, banner.SubTitle, banner.Description, banner.LinkTitle, banner.LinkURL,
		banner.ImageURL, banner.Position, banner.StartsAt, banner.EndsAt, banner.IsActive,
		banner.ID,
	).Scan(&banner.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrBannerNotFound
		}
		if isForeignKeyViolation(err) {
			return ErrPageNotFound
		}
		return fmt.Errorf("error updating banner: %w", err)
	}

	return nil
}

// Delete удаляет баннер по ID.
func (r *bannerRepository) Delete(id int) error {
	var deletedID int
	err := r.db.QueryRow(`DELETE FROM banners WHERE id = $1 RETURNING id`, id).Scan(&deletedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrBannerNotFound
		}
		return fmt.Errorf("error deleting banner: %w", err)
	}

	return nil
}

// isForeignKeyViolation сообщает, является ли err нарушением внешнего ключа Postgres
// (код 23503) — единственный способ отличить "page_id не существует" от прочих ошибок БД,
// так как в Go это не отдельный тип ошибки, а код в теле pq.Error.
func isForeignKeyViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23503"
	}
	return false
}
