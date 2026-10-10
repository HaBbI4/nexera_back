package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"nexera_back/internal/database"
	"nexera_back/internal/models"
)

// ErrServiceNotFound возвращается, когда услуги с указанным ID не существует — в том числе,
// когда service_id заявки ссылается на несуществующую услугу (нарушение внешнего ключа
// leads.service_id -> services.id, см. leadRepository.Create).
var ErrServiceNotFound = errors.New("service not found")

// ServiceSortField перечисляет поля, по которым разрешена сортировка списка услуг.
// Белый список по той же причине, что BannerSortField: колонку в ORDER BY нельзя
// параметризовать через $N, в SQL попадает только то, что есть в serviceSortColumns.
type ServiceSortField string

const (
	ServiceSortPosition  ServiceSortField = "position"
	ServiceSortPrice     ServiceSortField = "price"
	ServiceSortName      ServiceSortField = "name"
	ServiceSortCreatedAt ServiceSortField = "created_at"
)

// serviceSortColumns отображает ServiceSortField в реальную колонку таблицы services.
var serviceSortColumns = map[ServiceSortField]string{
	ServiceSortPosition:  "position",
	ServiceSortPrice:     "price",
	ServiceSortName:      "name",
	ServiceSortCreatedAt: "created_at",
}

// ServiceFilter описывает фильтрацию, сортировку и пагинацию списка услуг на уровне БД.
type ServiceFilter struct {
	// IsPublished — вернуть только опубликованные (true) или только скрытые (false) услуги;
	// nil — все. Публичный путь всегда передает true — решение об этом принимает сервис
	// (бизнес-правило), а не вызывающий код.
	IsPublished *bool
	// PriceFrom / PriceTo — границы цены включительно; nil — без ограничения. Услуги без
	// цены (price IS NULL, "цена по запросу") под ценовой фильтр не попадают никогда.
	PriceFrom *int
	PriceTo   *int

	Sort  ServiceSortField
	Order SortOrder

	Limit  int
	Offset int
}

// ServiceRepository описывает доступ к данным услуг компании.
type ServiceRepository interface {
	List(filter ServiceFilter) ([]*models.Service, error)
	GetByID(id int) (*models.Service, error)
	// Create сохраняет новую услугу, заполняя service.ID/CreatedAt/UpdatedAt значениями из БД.
	Create(service *models.Service) error
	// Update полностью перезаписывает услугу service.ID новым состоянием. Какие поля патчить,
	// а какие оставить как есть — решает сервис (см. ServiceService.Update).
	Update(service *models.Service) error
	// Delete удаляет услугу. Заявки на нее не удаляются: leads.service_id -> services(id)
	// ON DELETE SET NULL — у них просто пропадает ссылка на услугу.
	Delete(id int) error
}

// serviceRepository реализует ServiceRepository поверх PostgreSQL.
type serviceRepository struct {
	db *database.DB
}

// NewServiceRepository создает репозиторий услуг на PostgreSQL.
func NewServiceRepository(db *database.DB) ServiceRepository {
	return &serviceRepository{db: db}
}

// List возвращает услуги по фильтру, с сортировкой и пагинацией.
func (r *serviceRepository) List(filter ServiceFilter) ([]*models.Service, error) {
	query := `
		SELECT id, name, description, price, position, is_published, created_at, updated_at
		FROM services
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if filter.IsPublished != nil {
		query += fmt.Sprintf(" AND is_published = $%d", argIndex)
		args = append(args, *filter.IsPublished)
		argIndex++
	}

	if filter.PriceFrom != nil {
		query += fmt.Sprintf(" AND price >= $%d", argIndex)
		args = append(args, *filter.PriceFrom)
		argIndex++
	}

	if filter.PriceTo != nil {
		query += fmt.Sprintf(" AND price <= $%d", argIndex)
		args = append(args, *filter.PriceTo)
		argIndex++
	}

	column := serviceSortColumns[filter.Sort]
	if column == "" {
		column = serviceSortColumns[ServiceSortPosition]
	}
	direction := "ASC"
	if filter.Order == SortDesc {
		direction = "DESC"
	}
	// NULLS LAST — чтобы услуги без цены ("по запросу") при сортировке по price оказывались
	// в конце списка в обоих направлениях (по умолчанию Postgres ставит NULL первым при DESC).
	// Для NOT NULL колонок ни на что не влияет. id тем же направлением — чтобы строки
	// с одинаковым значением сортируемого поля не "плавали" между страницами пагинации.
	query += fmt.Sprintf(" ORDER BY %s %s NULLS LAST, id %s", column, direction, direction)

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
		return nil, fmt.Errorf("error getting services: %w", err)
	}
	defer rows.Close()

	var services []*models.Service
	for rows.Next() {
		service := &models.Service{}
		if err := rows.Scan(
			&service.ID, &service.Name, &service.Description, &service.Price, &service.Position,
			&service.IsPublished, &service.CreatedAt, &service.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan service: %w", err)
		}
		services = append(services, service)
	}

	return services, nil
}

// GetByID возвращает услугу по ID вне зависимости от того, опубликована ли она —
// решение, показывать ли ее, принимает сервис.
func (r *serviceRepository) GetByID(id int) (*models.Service, error) {
	service := &models.Service{}

	query := `
		SELECT id, name, description, price, position, is_published, created_at, updated_at
		FROM services
		WHERE id = $1
	`

	err := r.db.QueryRow(query, id).Scan(
		&service.ID, &service.Name, &service.Description, &service.Price, &service.Position,
		&service.IsPublished, &service.CreatedAt, &service.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrServiceNotFound
		}
		return nil, fmt.Errorf("error getting service: %w", err)
	}

	return service, nil
}

// Create сохраняет новую услугу.
func (r *serviceRepository) Create(service *models.Service) error {
	query := `
		INSERT INTO services (name, description, price, position, is_published)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		service.Name, service.Description, service.Price, service.Position, service.IsPublished,
	).Scan(&service.ID, &service.CreatedAt, &service.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error creating service: %w", err)
	}

	return nil
}

// Update перезаписывает услугу service.ID целиком новым состоянием и обновляет updated_at.
func (r *serviceRepository) Update(service *models.Service) error {
	query := `
		UPDATE services
		SET name = $1, description = $2, price = $3, position = $4, is_published = $5, updated_at = NOW()
		WHERE id = $6
		RETURNING updated_at
	`

	err := r.db.QueryRow(
		query,
		service.Name, service.Description, service.Price, service.Position, service.IsPublished,
		service.ID,
	).Scan(&service.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrServiceNotFound
		}
		return fmt.Errorf("error updating service: %w", err)
	}

	return nil
}

// Delete удаляет услугу по ID. leads.service_id -> services(id) ON DELETE SET NULL,
// поэтому заявки на эту услугу остаются, но без ссылки на нее.
func (r *serviceRepository) Delete(id int) error {
	var deletedID int
	err := r.db.QueryRow(`DELETE FROM services WHERE id = $1 RETURNING id`, id).Scan(&deletedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrServiceNotFound
		}
		return fmt.Errorf("error deleting service: %w", err)
	}

	return nil
}
