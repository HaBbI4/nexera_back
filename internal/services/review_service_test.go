package services_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"nexera_back/internal/models"
	"nexera_back/internal/repository"
	"nexera_back/internal/services"
)

// fakeReviewRepository — реализация repository.ReviewRepository в памяти. Запоминает
// последний переданный фильтр, чтобы тесты могли проверить, что сервис действительно
// применяет бизнес-правило видимости. Удаление мягкое, как в Postgres-реализации:
// удаленный отзыв остается в reviews, но для всех методов его больше нет.
type fakeReviewRepository struct {
	reviews           map[int]*models.Review
	lastFilter        repository.ReviewFilter
	lastSummaryFilter *bool
}

var _ repository.ReviewRepository = (*fakeReviewRepository)(nil)

func newFakeReviewRepository(reviews ...*models.Review) *fakeReviewRepository {
	repo := &fakeReviewRepository{reviews: make(map[int]*models.Review)}
	for _, r := range reviews {
		copied := *r
		repo.reviews[r.ID] = &copied
	}
	return repo
}

func (f *fakeReviewRepository) visible(id int) (*models.Review, bool) {
	review, ok := f.reviews[id]
	if !ok || review.DeletedAt != nil {
		return nil, false
	}
	return review, true
}

func (f *fakeReviewRepository) List(filter repository.ReviewFilter) ([]*models.Review, error) {
	f.lastFilter = filter

	var result []*models.Review
	for id := range f.reviews {
		review, ok := f.visible(id)
		if !ok {
			continue
		}
		if filter.IsApproved != nil && review.IsApproved != *filter.IsApproved {
			continue
		}
		copied := *review
		result = append(result, &copied)
	}
	return result, nil
}

func (f *fakeReviewRepository) GetByID(id int) (*models.Review, error) {
	review, ok := f.visible(id)
	if !ok {
		return nil, repository.ErrReviewNotFound
	}
	copied := *review
	return &copied, nil
}

func (f *fakeReviewRepository) Create(review *models.Review) error {
	review.ID = len(f.reviews) + 1
	review.IsApproved = false // DEFAULT FALSE в БД
	now := time.Now()
	review.CreatedAt = now
	review.UpdatedAt = now
	stored := *review
	f.reviews[review.ID] = &stored
	return nil
}

func (f *fakeReviewRepository) SetApproved(id int, approved bool) (*models.Review, error) {
	review, ok := f.visible(id)
	if !ok {
		return nil, repository.ErrReviewNotFound
	}
	review.IsApproved = approved
	copied := *review
	return &copied, nil
}

func (f *fakeReviewRepository) Delete(id int) error {
	review, ok := f.visible(id)
	if !ok {
		return repository.ErrReviewNotFound
	}
	now := time.Now()
	review.DeletedAt = &now
	return nil
}

func (f *fakeReviewRepository) GetSummary(isApproved *bool) (*models.ReviewSummary, error) {
	f.lastSummaryFilter = isApproved
	return &models.ReviewSummary{Count: 2, AverageRating: 4.5}, nil
}

// ExistsByPhoneSince, как и Postgres-реализация, учитывает и удаленные отзывы.
func (f *fakeReviewRepository) ExistsByPhoneSince(phone string, since time.Time) (bool, error) {
	for _, r := range f.reviews {
		if r.Phone == phone && !r.CreatedAt.Before(since) {
			return true, nil
		}
	}
	return false, nil
}

func newTestReview(id int, approved bool) *models.Review {
	return &models.Review{
		ID:         id,
		Name:       "Иван",
		Phone:      "+79991234567",
		Email:      ptr("ivan@example.com"),
		Rating:     5,
		Text:       "Отличный сайт сделали!",
		IsApproved: approved,
	}
}

func validReviewRequest() models.CreateReviewRequest {
	return models.CreateReviewRequest{
		Name:   "Иван",
		Phone:  "+79991234567",
		Rating: 5,
		Text:   "Отличный сайт сделали!",
	}
}

func TestReviewService_ListPublic_ForcesOnlyApproved(t *testing.T) {
	repo := newFakeReviewRepository(newTestReview(1, true), newTestReview(2, false))
	svc := services.NewReviewService(repo, 0, testLogger())

	list, err := svc.ListPublic(services.ListPublicReviewsFilter{
		RatingMin: ptr(4),
		Sort:      repository.ReviewSortRating,
		Order:     repository.SortDesc,
		Limit:     10,
		Offset:    5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(list) != 1 || list[0].ID != 1 {
		t.Errorf("expected only approved review #1, got %+v", list)
	}

	got := repo.lastFilter
	if got.IsApproved == nil || !*got.IsApproved || got.RatingMin == nil || *got.RatingMin != 4 ||
		got.Sort != repository.ReviewSortRating || got.Order != repository.SortDesc ||
		got.Limit != 10 || got.Offset != 5 {
		t.Errorf("filter not passed through correctly: %+v", got)
	}
}

func TestReviewService_ListPublic_ReturnsPublicFields(t *testing.T) {
	original := newTestReview(1, true)
	svc := services.NewReviewService(newFakeReviewRepository(original), 0, testLogger())

	list, err := svc.ListPublic(services.ListPublicReviewsFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Контактов в PublicReview нет на уровне типа; здесь проверяем, что остальные поля
	// перенесены, а не потерялись при преобразовании.
	got := list[0]
	if got.Name != original.Name || got.Rating != original.Rating || got.Text != original.Text {
		t.Errorf("public review fields not copied: %+v", got)
	}
}

func TestReviewService_ListPublic_HidesDeleted(t *testing.T) {
	repo := newFakeReviewRepository(newTestReview(1, true), newTestReview(2, true))
	svc := services.NewReviewService(repo, 0, testLogger())

	if err := svc.Delete(2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	list, err := svc.ListPublic(services.ListPublicReviewsFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].ID != 1 {
		t.Errorf("deleted review must not be listed, got %+v", list)
	}
}

func TestReviewService_GetPublicSummary_ForcesOnlyApproved(t *testing.T) {
	repo := newFakeReviewRepository()
	svc := services.NewReviewService(repo, 0, testLogger())

	if _, err := svc.GetPublicSummary(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastSummaryFilter == nil || !*repo.lastSummaryFilter {
		t.Errorf("summary must count only approved reviews, got filter %v", repo.lastSummaryFilter)
	}
}

func TestReviewService_List_DoesNotForceApproved(t *testing.T) {
	svc := services.NewReviewService(newFakeReviewRepository(newTestReview(1, true), newTestReview(2, false)), 0, testLogger())

	got, err := svc.List(services.ListAdminReviewsFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("admin List must return approved and unapproved, got %d items", len(got))
	}
}

func TestReviewService_List_FiltersUnapproved(t *testing.T) {
	svc := services.NewReviewService(newFakeReviewRepository(newTestReview(1, true), newTestReview(2, false)), 0, testLogger())

	got, err := svc.List(services.ListAdminReviewsFilter{IsApproved: ptr(false)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != 2 {
		t.Errorf("expected only unapproved #2, got %+v", got)
	}
}

func TestReviewService_GetByID_ReturnsContacts(t *testing.T) {
	svc := services.NewReviewService(newFakeReviewRepository(newTestReview(1, false)), 0, testLogger())

	got, err := svc.GetByID(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Phone == "" || got.Email == nil {
		t.Errorf("admin must see client contacts, got %+v", got)
	}
}

func TestReviewService_GetByID_NotFound(t *testing.T) {
	svc := services.NewReviewService(newFakeReviewRepository(), 0, testLogger())

	if _, err := svc.GetByID(99); !errors.Is(err, repository.ErrReviewNotFound) {
		t.Fatalf("expected ErrReviewNotFound, got %v", err)
	}
}

func TestReviewService_Create_NormalizesAndIsNotApproved(t *testing.T) {
	repo := newFakeReviewRepository()
	svc := services.NewReviewService(repo, 0, testLogger())

	review, err := svc.Create(models.CreateReviewRequest{
		Name:   "  Иван  ",
		Phone:  " +7 (999) 123-45-67 ",
		Email:  ptr("   "),
		Rating: 4,
		Text:   "  Хорошо  ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if review.Name != "Иван" || review.Text != "Хорошо" {
		t.Errorf("fields not trimmed: name=%q text=%q", review.Name, review.Text)
	}
	if review.Phone != "+79991234567" {
		t.Errorf("phone not normalized: %q", review.Phone)
	}
	if review.Email != nil {
		t.Errorf("empty email must become nil, got %q", *review.Email)
	}
	if review.IsApproved {
		t.Error("new review must not be approved")
	}
	if _, ok := repo.reviews[review.ID]; !ok {
		t.Error("review not saved")
	}
}

func TestReviewService_Create_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(r *models.CreateReviewRequest)
	}{
		{"empty name", func(r *models.CreateReviewRequest) { r.Name = "   " }},
		{"name too long", func(r *models.CreateReviewRequest) { r.Name = strings.Repeat("я", 256) }},
		{"empty phone", func(r *models.CreateReviewRequest) { r.Phone = "" }},
		{"phone without plus", func(r *models.CreateReviewRequest) { r.Phone = "89991234567" }},
		{"malformed email", func(r *models.CreateReviewRequest) { r.Email = ptr("not-an-email") }},
		{"email too long", func(r *models.CreateReviewRequest) { r.Email = ptr(strings.Repeat("a", 90) + "@example.com") }},
		{"rating zero", func(r *models.CreateReviewRequest) { r.Rating = 0 }},
		{"rating too high", func(r *models.CreateReviewRequest) { r.Rating = 6 }},
		{"empty text", func(r *models.CreateReviewRequest) { r.Text = "  " }},
		{"text too long", func(r *models.CreateReviewRequest) { r.Text = strings.Repeat("я", 5001) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeReviewRepository()
			svc := services.NewReviewService(repo, 0, testLogger())

			req := validReviewRequest()
			tt.mutate(&req)

			if _, err := svc.Create(req); !errors.Is(err, services.ErrInvalidReview) {
				t.Fatalf("expected ErrInvalidReview, got %v", err)
			}
			if len(repo.reviews) != 0 {
				t.Error("invalid review must not be saved")
			}
		})
	}
}

func TestReviewService_ApproveAndUnapprove(t *testing.T) {
	repo := newFakeReviewRepository(newTestReview(1, false))
	svc := services.NewReviewService(repo, 0, testLogger())

	review, err := svc.Approve(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !review.IsApproved || !repo.reviews[1].IsApproved {
		t.Error("review not approved")
	}

	// Повторное одобрение — не ошибка.
	if _, err := svc.Approve(1); err != nil {
		t.Fatalf("repeated approve must succeed, got %v", err)
	}

	review, err = svc.Unapprove(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if review.IsApproved || repo.reviews[1].IsApproved {
		t.Error("review approval not removed")
	}
}

func TestReviewService_Approve_NotFound(t *testing.T) {
	svc := services.NewReviewService(newFakeReviewRepository(), 0, testLogger())

	if _, err := svc.Approve(99); !errors.Is(err, repository.ErrReviewNotFound) {
		t.Fatalf("expected ErrReviewNotFound, got %v", err)
	}
	if _, err := svc.Unapprove(99); !errors.Is(err, repository.ErrReviewNotFound) {
		t.Fatalf("expected ErrReviewNotFound, got %v", err)
	}
}

func TestReviewService_Delete_IsSoftAndFinal(t *testing.T) {
	repo := newFakeReviewRepository(newTestReview(1, true))
	svc := services.NewReviewService(repo, 0, testLogger())

	if err := svc.Delete(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.reviews[1] == nil || repo.reviews[1].DeletedAt == nil {
		t.Fatal("review must be soft-deleted, not removed")
	}

	// Удаленный отзыв больше не существует ни для чтения, ни для модерации, ни для удаления.
	if _, err := svc.GetByID(1); !errors.Is(err, repository.ErrReviewNotFound) {
		t.Errorf("GetByID: expected ErrReviewNotFound, got %v", err)
	}
	if _, err := svc.Approve(1); !errors.Is(err, repository.ErrReviewNotFound) {
		t.Errorf("Approve: expected ErrReviewNotFound, got %v", err)
	}
	if err := svc.Delete(1); !errors.Is(err, repository.ErrReviewNotFound) {
		t.Errorf("Delete: expected ErrReviewNotFound, got %v", err)
	}
}

func TestReviewService_Delete_NotFound(t *testing.T) {
	svc := services.NewReviewService(newFakeReviewRepository(), 0, testLogger())

	if err := svc.Delete(99); !errors.Is(err, repository.ErrReviewNotFound) {
		t.Fatalf("expected ErrReviewNotFound, got %v", err)
	}
}

func TestReviewService_Create_HoneypotIsNotSaved(t *testing.T) {
	repo := newFakeReviewRepository()
	svc := services.NewReviewService(repo, 0, testLogger())

	req := validReviewRequest()
	req.Website = "http://spam.example"

	if _, err := svc.Create(req); !errors.Is(err, services.ErrSpamDetected) {
		t.Fatalf("expected ErrSpamDetected, got %v", err)
	}
	if len(repo.reviews) != 0 {
		t.Error("spam review must not be saved")
	}
}

func TestReviewService_Create_DuplicatePhoneWithinWindow(t *testing.T) {
	existing := newTestReview(1, false)
	existing.CreatedAt = time.Now().Add(-time.Hour)
	repo := newFakeReviewRepository(existing)
	svc := services.NewReviewService(repo, 24*time.Hour, testLogger())

	if _, err := svc.Create(validReviewRequest()); !errors.Is(err, services.ErrDuplicateSubmission) {
		t.Fatalf("expected ErrDuplicateSubmission, got %v", err)
	}
	if len(repo.reviews) != 1 {
		t.Error("duplicate review must not be saved")
	}
}

func TestReviewService_Create_DuplicateOfDeletedReview(t *testing.T) {
	existing := newTestReview(1, false)
	existing.CreatedAt = time.Now().Add(-time.Hour)
	repo := newFakeReviewRepository(existing)
	svc := services.NewReviewService(repo, 24*time.Hour, testLogger())

	// Удаленный админом спам не открывает дорогу следующему отзыву с того же телефона.
	if err := svc.Delete(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := svc.Create(validReviewRequest()); !errors.Is(err, services.ErrDuplicateSubmission) {
		t.Fatalf("expected ErrDuplicateSubmission, got %v", err)
	}
}

func TestReviewService_Create_SamePhoneOutsideWindow(t *testing.T) {
	existing := newTestReview(1, true)
	existing.CreatedAt = time.Now().Add(-48 * time.Hour)
	svc := services.NewReviewService(newFakeReviewRepository(existing), 24*time.Hour, testLogger())

	if _, err := svc.Create(validReviewRequest()); err != nil {
		t.Fatalf("review outside duplicate window must be accepted, got %v", err)
	}
}
