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

// fakeFAQRepository — реализация repository.FAQRepository в памяти. Запоминает последний
// переданный фильтр, чтобы тесты могли проверить, что сервис действительно применяет
// бизнес-правило видимости. GetByID возвращает копию — изменения сервиса не "сохраняются"
// без вызова Update.
type fakeFAQRepository struct {
	faqs       map[int]*models.FAQ
	lastFilter repository.FAQFilter
}

var _ repository.FAQRepository = (*fakeFAQRepository)(nil)

func newFakeFAQRepository(faqs ...*models.FAQ) *fakeFAQRepository {
	repo := &fakeFAQRepository{faqs: make(map[int]*models.FAQ)}
	for _, f := range faqs {
		copied := *f
		repo.faqs[f.ID] = &copied
	}
	return repo
}

func (f *fakeFAQRepository) List(filter repository.FAQFilter) ([]*models.FAQ, error) {
	f.lastFilter = filter

	var result []*models.FAQ
	for _, faq := range f.faqs {
		if filter.IsPublished != nil && faq.IsPublished != *filter.IsPublished {
			continue
		}
		result = append(result, faq)
	}
	return result, nil
}

func (f *fakeFAQRepository) GetByID(id int) (*models.FAQ, error) {
	faq, ok := f.faqs[id]
	if !ok {
		return nil, repository.ErrFAQNotFound
	}
	copied := *faq
	return &copied, nil
}

func (f *fakeFAQRepository) Create(faq *models.FAQ) error {
	faq.ID = len(f.faqs) + 1
	now := time.Now()
	faq.CreatedAt = now
	faq.UpdatedAt = now
	stored := *faq
	f.faqs[faq.ID] = &stored
	return nil
}

func (f *fakeFAQRepository) Update(faq *models.FAQ) error {
	if _, ok := f.faqs[faq.ID]; !ok {
		return repository.ErrFAQNotFound
	}
	faq.UpdatedAt = time.Now()
	stored := *faq
	f.faqs[faq.ID] = &stored
	return nil
}

func (f *fakeFAQRepository) Delete(id int) error {
	if _, ok := f.faqs[id]; !ok {
		return repository.ErrFAQNotFound
	}
	delete(f.faqs, id)
	return nil
}

func newTestFAQ(id int, published bool) *models.FAQ {
	return &models.FAQ{
		ID:          id,
		Question:    "Сколько стоит сайт?",
		Answer:      "Зависит от задачи.",
		Position:    id,
		IsPublished: published,
	}
}

func TestFAQService_ListPublic_ForcesOnlyPublished(t *testing.T) {
	repo := newFakeFAQRepository(newTestFAQ(1, true), newTestFAQ(2, false))
	svc := services.NewFAQService(repo, testLogger())

	faqs, err := svc.ListPublic(services.ListFAQFilter{
		Sort:   repository.FAQSortCreatedAt,
		Order:  repository.SortDesc,
		Limit:  10,
		Offset: 5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(faqs) != 1 || faqs[0].ID != 1 {
		t.Errorf("expected only published faq #1, got %+v", faqs)
	}

	got := repo.lastFilter
	if got.IsPublished == nil || !*got.IsPublished || got.Sort != repository.FAQSortCreatedAt || got.Order != repository.SortDesc ||
		got.Limit != 10 || got.Offset != 5 {
		t.Errorf("filter not passed through correctly: %+v", got)
	}
}

func TestFAQService_GetPublicByID(t *testing.T) {
	svc := services.NewFAQService(newFakeFAQRepository(newTestFAQ(1, true)), testLogger())

	faq, err := svc.GetPublicByID(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if faq.ID != 1 {
		t.Errorf("got ID %d, want 1", faq.ID)
	}
}

func TestFAQService_GetPublicByID_UnpublishedIsNotFound(t *testing.T) {
	svc := services.NewFAQService(newFakeFAQRepository(newTestFAQ(1, false)), testLogger())

	if _, err := svc.GetPublicByID(1); !errors.Is(err, repository.ErrFAQNotFound) {
		t.Fatalf("expected ErrFAQNotFound, got %v", err)
	}
}

func TestFAQService_GetPublicByID_NotFound(t *testing.T) {
	svc := services.NewFAQService(newFakeFAQRepository(), testLogger())

	if _, err := svc.GetPublicByID(99); !errors.Is(err, repository.ErrFAQNotFound) {
		t.Fatalf("expected ErrFAQNotFound, got %v", err)
	}
}

func TestFAQService_Create(t *testing.T) {
	repo := newFakeFAQRepository()
	svc := services.NewFAQService(repo, testLogger())

	faq, err := svc.Create(models.CreateFAQRequest{
		Question:    "Сколько стоит сайт?",
		Answer:      "Зависит от задачи.",
		Position:    3,
		IsPublished: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if faq.ID == 0 {
		t.Error("faq ID not assigned")
	}
	if _, ok := repo.faqs[faq.ID]; !ok {
		t.Error("faq not saved")
	}
}

func TestFAQService_Create_Invalid(t *testing.T) {
	valid := models.CreateFAQRequest{Question: "Вопрос?", Answer: "Ответ."}

	tests := []struct {
		name   string
		mutate func(r *models.CreateFAQRequest)
	}{
		{"empty question", func(r *models.CreateFAQRequest) { r.Question = "" }},
		{"question too long", func(r *models.CreateFAQRequest) { r.Question = strings.Repeat("я", 256) }},
		{"empty answer", func(r *models.CreateFAQRequest) { r.Answer = "" }},
		{"answer too long", func(r *models.CreateFAQRequest) { r.Answer = strings.Repeat("я", 5001) }},
		{"negative position", func(r *models.CreateFAQRequest) { r.Position = -1 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeFAQRepository()
			svc := services.NewFAQService(repo, testLogger())

			req := valid
			tt.mutate(&req)

			if _, err := svc.Create(req); !errors.Is(err, services.ErrInvalidFAQ) {
				t.Fatalf("expected ErrInvalidFAQ, got %v", err)
			}
			if len(repo.faqs) != 0 {
				t.Error("invalid faq must not be saved")
			}
		})
	}
}

func TestFAQService_Create_MaxLengthIsCountedInCharacters(t *testing.T) {
	svc := services.NewFAQService(newFakeFAQRepository(), testLogger())

	// 255 кириллических символов — это 510 байт, но VARCHAR(255) считает символы.
	_, err := svc.Create(models.CreateFAQRequest{Question: strings.Repeat("я", 255), Answer: "Ответ."})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFAQService_Update_PatchesOnlyProvidedFields(t *testing.T) {
	original := newTestFAQ(1, true)
	svc := services.NewFAQService(newFakeFAQRepository(original), testLogger())

	updated, err := svc.Update(1, models.UpdateFAQRequest{IsPublished: ptr(false)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.IsPublished {
		t.Error("is_published not updated")
	}
	if updated.Question != original.Question || updated.Answer != original.Answer || updated.Position != original.Position {
		t.Errorf("unchanged fields were modified: %+v", updated)
	}
}

func TestFAQService_Update_InvalidMergedFAQ(t *testing.T) {
	original := newTestFAQ(1, true)
	repo := newFakeFAQRepository(original)
	svc := services.NewFAQService(repo, testLogger())

	tests := []struct {
		name string
		req  models.UpdateFAQRequest
	}{
		{"empty question", models.UpdateFAQRequest{Question: ptr("")}},
		{"empty answer", models.UpdateFAQRequest{Answer: ptr("")}},
		{"negative position", models.UpdateFAQRequest{Position: ptr(-5)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.Update(1, tt.req); !errors.Is(err, services.ErrInvalidFAQ) {
				t.Fatalf("expected ErrInvalidFAQ, got %v", err)
			}
			if *repo.faqs[1] != *original {
				t.Error("invalid update must not be saved")
			}
		})
	}
}

func TestFAQService_Update_NotFound(t *testing.T) {
	svc := services.NewFAQService(newFakeFAQRepository(), testLogger())

	if _, err := svc.Update(99, models.UpdateFAQRequest{Answer: ptr("Ответ.")}); !errors.Is(err, repository.ErrFAQNotFound) {
		t.Fatalf("expected ErrFAQNotFound, got %v", err)
	}
}

func TestFAQService_Delete(t *testing.T) {
	repo := newFakeFAQRepository(newTestFAQ(1, true))
	svc := services.NewFAQService(repo, testLogger())

	if err := svc.Delete(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := repo.faqs[1]; ok {
		t.Error("faq was not deleted")
	}
}

func TestFAQService_Delete_NotFound(t *testing.T) {
	svc := services.NewFAQService(newFakeFAQRepository(), testLogger())

	if err := svc.Delete(99); !errors.Is(err, repository.ErrFAQNotFound) {
		t.Fatalf("expected ErrFAQNotFound, got %v", err)
	}
}

func TestFAQService_List_DoesNotForcePublished(t *testing.T) {
	svc := services.NewFAQService(newFakeFAQRepository(newTestFAQ(1, true), newTestFAQ(2, false)), testLogger())

	got, err := svc.List(services.ListAdminFAQFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("admin List must return published and unpublished, got %d items", len(got))
	}
}

func TestFAQService_List_FiltersUnpublished(t *testing.T) {
	svc := services.NewFAQService(newFakeFAQRepository(newTestFAQ(1, true), newTestFAQ(2, false)), testLogger())

	got, err := svc.List(services.ListAdminFAQFilter{IsPublished: ptr(false)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != 2 {
		t.Errorf("expected only unpublished #2, got %+v", got)
	}
}

func TestFAQService_GetByID_ReturnsUnpublished(t *testing.T) {
	svc := services.NewFAQService(newFakeFAQRepository(newTestFAQ(1, false)), testLogger())

	got, err := svc.GetByID(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.IsPublished {
		t.Error("expected the unpublished item")
	}
}

func TestFAQService_GetByID_NotFound(t *testing.T) {
	svc := services.NewFAQService(newFakeFAQRepository(), testLogger())

	if _, err := svc.GetByID(99); !errors.Is(err, repository.ErrFAQNotFound) {
		t.Fatalf("expected ErrFAQNotFound, got %v", err)
	}
}
