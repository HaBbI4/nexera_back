package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"nexera_back/internal/database"
	"nexera_back/internal/models"
)

// ErrLeadNotFound возвращается, когда заявки с указанным ID не существует.
// Если service_id заявки ссылается на несуществующую услугу (нарушение внешнего ключа
// leads.service_id -> services.id), возвращается ErrServiceNotFound из service_repository.go.
var ErrLeadNotFound = errors.New("lead not found")

// LeadSortField перечисляет поля, по которым разрешена сортировка списка заявок.
// Белый список по той же причине, что BannerSortField: колонку в ORDER BY нельзя
// параметризовать через $N, в SQL попадает только то, что есть в leadSortColumns.
type LeadSortField string

const (
	LeadSortCreatedAt LeadSortField = "created_at"
	LeadSortName      LeadSortField = "name"
)

// leadSortColumns отображает LeadSortField в реальную колонку таблицы leads. Колонки
// с префиксом l. — в запросе есть JOIN с services, где тоже есть name/created_at.
var leadSortColumns = map[LeadSortField]string{
	LeadSortCreatedAt: "l.created_at",
	LeadSortName:      "l.name",
}

// LeadFilter описывает фильтрацию, сортировку и пагинацию списка заявок на уровне БД.
type LeadFilter struct {
	// ServiceID — вернуть только заявки на эту услугу; nil — на любую (и без услуги).
	ServiceID *int
	// Source — вернуть только заявки из этого источника (contact_form и т.п.); nil — из любого.
	Source *string
	// CreatedFrom / CreatedTo — границы created_at включительно; nil — без ограничения.
	CreatedFrom *time.Time
	CreatedTo   *time.Time

	Sort  LeadSortField
	Order SortOrder

	Limit  int
	Offset int
}

// LeadRepository описывает доступ к данным заявок клиентов.
type LeadRepository interface {
	// List и GetByID возвращают заявку вместе с услугой (lead.Service), если она указана
	// и еще существует; иначе lead.Service == nil.
	List(filter LeadFilter) ([]*models.Lead, error)
	GetByID(id int) (*models.Lead, error)
	// Create сохраняет новую заявку, заполняя lead.ID/CreatedAt/UpdatedAt значениями из БД.
	// lead.Service не заполняется.
	Create(lead *models.Lead) error
	Delete(id int) error
	// ExistsByPhoneSince сообщает, была ли заявка с этим телефоном, созданная не раньше since.
	ExistsByPhoneSince(phone string, since time.Time) (bool, error)
}

// leadRepository реализует LeadRepository поверх PostgreSQL.
type leadRepository struct {
	db *database.DB
}

// NewLeadRepository создает репозиторий заявок на PostgreSQL.
func NewLeadRepository(db *database.DB) LeadRepository {
	return &leadRepository{db: db}
}

// leadSelect — общая часть SELECT для чтения заявки вместе с ее услугой; порядок колонок
// соответствует scanLeadRow. LEFT JOIN — заявка может быть без услуги (service_id IS NULL
// изначально или обнулен ON DELETE SET NULL после удаления услуги).
const leadSelect = `
	SELECT l.id, l.service_id, l.name, l.phone, l.email, l.message, l.source,
	       l.created_at, l.updated_at,
	       s.id, s.name, s.description, s.price, s.position, s.is_published,
	       s.created_at, s.updated_at
	FROM leads l
	LEFT JOIN services s ON s.id = l.service_id
`

func scanLeadRow(row rowScanner) (*models.Lead, error) {
	lead := &models.Lead{}

	// Колонки услуги после LEFT JOIN все NULL, если услуги нет, поэтому сканируются
	// в nullable-обертки, а не напрямую в models.Service.
	var (
		serviceID          sql.NullInt64
		serviceName        sql.NullString
		serviceDescription sql.NullString
		servicePrice       sql.NullInt64
		servicePosition    sql.NullInt64
		serviceIsPublished sql.NullBool
		serviceCreatedAt   sql.NullTime
		serviceUpdatedAt   sql.NullTime
	)

	err := row.Scan(
		&lead.ID, &lead.ServiceID, &lead.Name, &lead.Phone, &lead.Email, &lead.Message, &lead.Source,
		&lead.CreatedAt, &lead.UpdatedAt,
		&serviceID, &serviceName, &serviceDescription, &servicePrice, &servicePosition, &serviceIsPublished,
		&serviceCreatedAt, &serviceUpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if serviceID.Valid {
		lead.Service = &models.Service{
			ID:          int(serviceID.Int64),
			Name:        serviceName.String,
			Position:    int(servicePosition.Int64),
			IsPublished: serviceIsPublished.Bool,
			CreatedAt:   serviceCreatedAt.Time,
			UpdatedAt:   serviceUpdatedAt.Time,
		}
		if serviceDescription.Valid {
			lead.Service.Description = &serviceDescription.String
		}
		if servicePrice.Valid {
			price := int(servicePrice.Int64)
			lead.Service.Price = &price
		}
	}

	return lead, nil
}

// List возвращает заявки (вместе с услугами) по фильтру, с сортировкой и пагинацией.
func (r *leadRepository) List(filter LeadFilter) ([]*models.Lead, error) {
	query := leadSelect + ` WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if filter.ServiceID != nil {
		query += fmt.Sprintf(" AND l.service_id = $%d", argIndex)
		args = append(args, *filter.ServiceID)
		argIndex++
	}

	if filter.Source != nil {
		query += fmt.Sprintf(" AND l.source = $%d", argIndex)
		args = append(args, *filter.Source)
		argIndex++
	}

	if filter.CreatedFrom != nil {
		query += fmt.Sprintf(" AND l.created_at >= $%d", argIndex)
		args = append(args, *filter.CreatedFrom)
		argIndex++
	}

	if filter.CreatedTo != nil {
		query += fmt.Sprintf(" AND l.created_at <= $%d", argIndex)
		args = append(args, *filter.CreatedTo)
		argIndex++
	}

	column := leadSortColumns[filter.Sort]
	if column == "" {
		column = leadSortColumns[LeadSortCreatedAt]
	}
	direction := "ASC"
	if filter.Order == SortDesc {
		direction = "DESC"
	}
	// id тем же направлением — чтобы строки с одинаковым значением сортируемого поля
	// не "плавали" между запросами страниц пагинации.
	query += fmt.Sprintf(" ORDER BY %s %s, l.id %s", column, direction, direction)

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
		return nil, fmt.Errorf("error getting leads: %w", err)
	}
	defer rows.Close()

	var leads []*models.Lead
	for rows.Next() {
		lead, err := scanLeadRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lead: %w", err)
		}
		leads = append(leads, lead)
	}

	return leads, nil
}

// GetByID возвращает заявку (вместе с услугой) по ID.
func (r *leadRepository) GetByID(id int) (*models.Lead, error) {
	lead, err := scanLeadRow(r.db.QueryRow(leadSelect+` WHERE l.id = $1`, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrLeadNotFound
		}
		return nil, fmt.Errorf("error getting lead: %w", err)
	}

	return lead, nil
}

// Create сохраняет новую заявку.
func (r *leadRepository) Create(lead *models.Lead) error {
	query := `
		INSERT INTO leads (service_id, name, phone, email, message, source)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(query, lead.ServiceID, lead.Name, lead.Phone, lead.Email, lead.Message, lead.Source).
		Scan(&lead.ID, &lead.CreatedAt, &lead.UpdatedAt)
	if err != nil {
		if isForeignKeyViolation(err) {
			return ErrServiceNotFound
		}
		return fmt.Errorf("error creating lead: %w", err)
	}

	return nil
}

// Delete удаляет заявку по ID.
func (r *leadRepository) Delete(id int) error {
	var deletedID int
	err := r.db.QueryRow(`DELETE FROM leads WHERE id = $1 RETURNING id`, id).Scan(&deletedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrLeadNotFound
		}
		return fmt.Errorf("error deleting lead: %w", err)
	}

	return nil
}

// ExistsByPhoneSince сообщает, была ли заявка с этим телефоном, созданная не раньше since.
func (r *leadRepository) ExistsByPhoneSince(phone string, since time.Time) (bool, error) {
	var exists bool
	err := r.db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM leads WHERE phone = $1 AND created_at >= $2)`,
		phone, since,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("error checking recent lead by phone: %w", err)
	}

	return exists, nil
}
