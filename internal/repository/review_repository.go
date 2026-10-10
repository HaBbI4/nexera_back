package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"nexera_back/internal/database"
	"nexera_back/internal/models"
)

// ErrReviewNotFound возвращается, когда отзыва с указанным ID нет или он удален
// (мягкое удаление: deleted_at IS NOT NULL).
var ErrReviewNotFound = errors.New("review not found")

// ReviewSortField перечисляет поля, по которым разрешена сортировка списка отзывов.
// Белый список по той же причине, что BannerSortField: колонку в ORDER BY нельзя
// параметризовать через $N, в SQL попадает только то, что есть в reviewSortColumns.
type ReviewSortField string

const (
	ReviewSortCreatedAt ReviewSortField = "created_at"
	ReviewSortRating    ReviewSortField = "rating"
)

// reviewSortColumns отображает ReviewSortField в реальную колонку таблицы reviews.
var reviewSortColumns = map[ReviewSortField]string{
	ReviewSortCreatedAt: "created_at",
	ReviewSortRating:    "rating",
}

// ReviewFilter описывает фильтрацию, сортировку и пагинацию списка отзывов на уровне БД.
// Удаленные отзывы не возвращаются никогда, независимо от фильтра.
type ReviewFilter struct {
	// IsApproved — вернуть только одобренные (true) или только неодобренные (false);
	// nil — любые.
	IsApproved *bool
	// RatingMin — вернуть только отзывы с оценкой не ниже; nil — с любой.
	RatingMin *int
	// CreatedFrom / CreatedTo — границы created_at включительно; nil — без ограничения.
	CreatedFrom *time.Time
	CreatedTo   *time.Time

	Sort  ReviewSortField
	Order SortOrder

	Limit  int
	Offset int
}

// ReviewRepository описывает доступ к данным отзывов клиентов. Все методы работают только
// с неудаленными отзывами: удаленный отзыв для них не существует (ErrReviewNotFound).
type ReviewRepository interface {
	List(filter ReviewFilter) ([]*models.Review, error)
	GetByID(id int) (*models.Review, error)
	// Create сохраняет новый отзыв, заполняя review.ID/IsApproved/CreatedAt/UpdatedAt
	// значениями из БД.
	Create(review *models.Review) error
	// SetApproved одобряет отзыв или снимает одобрение и возвращает его новое состояние.
	SetApproved(id int, approved bool) (*models.Review, error)
	// Delete мягко удаляет отзыв (deleted_at = NOW()).
	Delete(id int) error
	// GetSummary возвращает количество и средний рейтинг отзывов; isApproved — как в ReviewFilter.
	GetSummary(isApproved *bool) (*models.ReviewSummary, error)
	// ExistsByPhoneSince сообщает, был ли отзыв с этим телефоном, созданный не раньше since.
	// Единственный метод, который учитывает и удаленные отзывы: удаленный админом спам
	// не должен открывать дорогу следующему.
	ExistsByPhoneSince(phone string, since time.Time) (bool, error)
}

// reviewRepository реализует ReviewRepository поверх PostgreSQL.
type reviewRepository struct {
	db *database.DB
}

// NewReviewRepository создает репозиторий отзывов на PostgreSQL.
func NewReviewRepository(db *database.DB) ReviewRepository {
	return &reviewRepository{db: db}
}

// reviewColumns — общий список колонок отзыва; порядок соответствует scanReviewRow.
const reviewColumns = `id, name, phone, email, rating, text, is_approved, created_at, updated_at`

func scanReviewRow(row rowScanner) (*models.Review, error) {
	review := &models.Review{}
	err := row.Scan(
		&review.ID, &review.Name, &review.Phone, &review.Email, &review.Rating, &review.Text,
		&review.IsApproved, &review.CreatedAt, &review.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return review, nil
}

// List возвращает неудаленные отзывы по фильтру, с сортировкой и пагинацией.
func (r *reviewRepository) List(filter ReviewFilter) ([]*models.Review, error) {
	query := `SELECT ` + reviewColumns + ` FROM reviews WHERE deleted_at IS NULL`
	args := []interface{}{}
	argIndex := 1

	if filter.IsApproved != nil {
		query += fmt.Sprintf(" AND is_approved = $%d", argIndex)
		args = append(args, *filter.IsApproved)
		argIndex++
	}

	if filter.RatingMin != nil {
		query += fmt.Sprintf(" AND rating >= $%d", argIndex)
		args = append(args, *filter.RatingMin)
		argIndex++
	}

	if filter.CreatedFrom != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *filter.CreatedFrom)
		argIndex++
	}

	if filter.CreatedTo != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *filter.CreatedTo)
		argIndex++
	}

	column := reviewSortColumns[filter.Sort]
	if column == "" {
		column = reviewSortColumns[ReviewSortCreatedAt]
	}
	direction := "ASC"
	if filter.Order == SortDesc {
		direction = "DESC"
	}
	// id тем же направлением — чтобы строки с одинаковым значением сортируемого поля
	// (например, много отзывов с rating = 5) не "плавали" между страницами пагинации.
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
		return nil, fmt.Errorf("error getting reviews: %w", err)
	}
	defer rows.Close()

	var reviews []*models.Review
	for rows.Next() {
		review, err := scanReviewRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan review: %w", err)
		}
		reviews = append(reviews, review)
	}

	return reviews, nil
}

// GetByID возвращает неудаленный отзыв по ID.
func (r *reviewRepository) GetByID(id int) (*models.Review, error) {
	query := `SELECT ` + reviewColumns + ` FROM reviews WHERE id = $1 AND deleted_at IS NULL`

	review, err := scanReviewRow(r.db.QueryRow(query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrReviewNotFound
		}
		return nil, fmt.Errorf("error getting review: %w", err)
	}

	return review, nil
}

// Create сохраняет новый отзыв. is_approved не передается — берется DEFAULT FALSE из БД:
// новый отзыв всегда ждет модерации.
func (r *reviewRepository) Create(review *models.Review) error {
	query := `
		INSERT INTO reviews (name, phone, email, rating, text)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, is_approved, created_at, updated_at
	`

	err := r.db.QueryRow(query, review.Name, review.Phone, review.Email, review.Rating, review.Text).
		Scan(&review.ID, &review.IsApproved, &review.CreatedAt, &review.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error creating review: %w", err)
	}

	return nil
}

// SetApproved одобряет отзыв или снимает одобрение. updated_at меняется, только если
// флаг действительно изменился — повторный approve ничего не меняет.
func (r *reviewRepository) SetApproved(id int, approved bool) (*models.Review, error) {
	query := `
		UPDATE reviews
		SET is_approved = $2,
		    updated_at = CASE WHEN is_approved = $2 THEN updated_at ELSE NOW() END
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING ` + reviewColumns

	review, err := scanReviewRow(r.db.QueryRow(query, id, approved))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrReviewNotFound
		}
		return nil, fmt.Errorf("error setting review approval: %w", err)
	}

	return review, nil
}

// Delete мягко удаляет отзыв: строка остается в БД (deleted_at = NOW()), но для
// остальных методов репозитория ее больше нет.
func (r *reviewRepository) Delete(id int) error {
	var deletedID int
	err := r.db.QueryRow(
		`UPDATE reviews SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL RETURNING id`,
		id,
	).Scan(&deletedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrReviewNotFound
		}
		return fmt.Errorf("error deleting review: %w", err)
	}

	return nil
}

// GetSummary возвращает количество и средний рейтинг неудаленных отзывов.
func (r *reviewRepository) GetSummary(isApproved *bool) (*models.ReviewSummary, error) {
	query := `
		SELECT COUNT(*), COALESCE(ROUND(AVG(rating), 1), 0)::float8
		FROM reviews
		WHERE deleted_at IS NULL
	`
	args := []interface{}{}

	if isApproved != nil {
		query += ` AND is_approved = $1`
		args = append(args, *isApproved)
	}

	summary := &models.ReviewSummary{}
	if err := r.db.QueryRow(query, args...).Scan(&summary.Count, &summary.AverageRating); err != nil {
		return nil, fmt.Errorf("error getting review summary: %w", err)
	}

	return summary, nil
}

// ExistsByPhoneSince сообщает, был ли отзыв с этим телефоном, созданный не раньше since,
// включая удаленные.
func (r *reviewRepository) ExistsByPhoneSince(phone string, since time.Time) (bool, error) {
	var exists bool
	err := r.db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM reviews WHERE phone = $1 AND created_at >= $2)`,
		phone, since,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("error checking recent review by phone: %w", err)
	}

	return exists, nil
}
