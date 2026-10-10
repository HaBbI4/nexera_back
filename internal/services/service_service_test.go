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

// fakeServiceRepository — реализация repository.ServiceRepository в памяти. Запоминает
// последний переданный фильтр, чтобы тесты могли проверить, что сервис действительно
// применяет бизнес-правило видимости. GetByID возвращает копию — изменения сервиса
// не "сохраняются" без вызова Update.
type fakeServiceRepository struct {
	services   map[int]*models.Service
	lastFilter repository.ServiceFilter
}

var _ repository.ServiceRepository = (*fakeServiceRepository)(nil)

func newFakeServiceRepository(items ...*models.Service) *fakeServiceRepository {
	repo := &fakeServiceRepository{services: make(map[int]*models.Service)}
	for _, s := range items {
		copied := *s
		repo.services[s.ID] = &copied
	}
	return repo
}

func (f *fakeServiceRepository) List(filter repository.ServiceFilter) ([]*models.Service, error) {
	f.lastFilter = filter

	var result []*models.Service
	for _, s := range f.services {
		if filter.IsPublished != nil && s.IsPublished != *filter.IsPublished {
			continue
		}
		result = append(result, s)
	}
	return result, nil
}

func (f *fakeServiceRepository) GetByID(id int) (*models.Service, error) {
	service, ok := f.services[id]
	if !ok {
		return nil, repository.ErrServiceNotFound
	}
	copied := *service
	return &copied, nil
}

func (f *fakeServiceRepository) Create(service *models.Service) error {
	service.ID = len(f.services) + 1
	now := time.Now()
	service.CreatedAt = now
	service.UpdatedAt = now
	stored := *service
	f.services[service.ID] = &stored
	return nil
}

func (f *fakeServiceRepository) Update(service *models.Service) error {
	if _, ok := f.services[service.ID]; !ok {
		return repository.ErrServiceNotFound
	}
	service.UpdatedAt = time.Now()
	stored := *service
	f.services[service.ID] = &stored
	return nil
}

func (f *fakeServiceRepository) Delete(id int) error {
	if _, ok := f.services[id]; !ok {
		return repository.ErrServiceNotFound
	}
	delete(f.services, id)
	return nil
}

func newTestService(id int, published bool) *models.Service {
	return &models.Service{
		ID:          id,
		Name:        "Лендинг",
		Description: ptr("Одностраничный сайт"),
		Price:       ptr(50000),
		Position:    id,
		IsPublished: published,
	}
}

func TestServiceService_ListPublic_ForcesOnlyPublished(t *testing.T) {
	repo := newFakeServiceRepository(newTestService(1, true), newTestService(2, false))
	svc := services.NewServiceService(repo, testLogger())

	list, err := svc.ListPublic(services.ListServicesFilter{
		PriceFrom: ptr(1000),
		PriceTo:   ptr(90000),
		Sort:      repository.ServiceSortPrice,
		Order:     repository.SortDesc,
		Limit:     10,
		Offset:    5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(list) != 1 || list[0].ID != 1 {
		t.Errorf("expected only published service #1, got %+v", list)
	}

	got := repo.lastFilter
	if got.IsPublished == nil || !*got.IsPublished || got.PriceFrom == nil || *got.PriceFrom != 1000 || got.PriceTo == nil || *got.PriceTo != 90000 ||
		got.Sort != repository.ServiceSortPrice || got.Order != repository.SortDesc ||
		got.Limit != 10 || got.Offset != 5 {
		t.Errorf("filter not passed through correctly: %+v", got)
	}
}

func TestServiceService_GetPublicByID(t *testing.T) {
	svc := services.NewServiceService(newFakeServiceRepository(newTestService(1, true)), testLogger())

	service, err := svc.GetPublicByID(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if service.ID != 1 {
		t.Errorf("got ID %d, want 1", service.ID)
	}
}

func TestServiceService_GetPublicByID_UnpublishedIsNotFound(t *testing.T) {
	svc := services.NewServiceService(newFakeServiceRepository(newTestService(1, false)), testLogger())

	if _, err := svc.GetPublicByID(1); !errors.Is(err, repository.ErrServiceNotFound) {
		t.Fatalf("expected ErrServiceNotFound, got %v", err)
	}
}

func TestServiceService_GetPublicByID_NotFound(t *testing.T) {
	svc := services.NewServiceService(newFakeServiceRepository(), testLogger())

	if _, err := svc.GetPublicByID(99); !errors.Is(err, repository.ErrServiceNotFound) {
		t.Fatalf("expected ErrServiceNotFound, got %v", err)
	}
}

func TestServiceService_Create(t *testing.T) {
	repo := newFakeServiceRepository()
	svc := services.NewServiceService(repo, testLogger())

	service, err := svc.Create(models.CreateServiceRequest{
		Name:        "Интернет-магазин",
		Description: ptr("Каталог, корзина, оплата"),
		Price:       ptr(150000),
		Position:    2,
		IsPublished: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if service.ID == 0 {
		t.Error("service ID not assigned")
	}
	if _, ok := repo.services[service.ID]; !ok {
		t.Error("service not saved")
	}
}

func TestServiceService_Create_WithoutPrice(t *testing.T) {
	svc := services.NewServiceService(newFakeServiceRepository(), testLogger())

	// Без цены — "цена по запросу", это валидная услуга.
	service, err := svc.Create(models.CreateServiceRequest{Name: "Индивидуальная разработка"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if service.Price != nil {
		t.Errorf("price must stay nil, got %d", *service.Price)
	}
}

func TestServiceService_Create_Invalid(t *testing.T) {
	tests := []struct {
		name string
		req  models.CreateServiceRequest
	}{
		{"empty name", models.CreateServiceRequest{Name: ""}},
		{"name too long", models.CreateServiceRequest{Name: strings.Repeat("я", 101)}},
		{"description too long", models.CreateServiceRequest{Name: "Сайт", Description: ptr(strings.Repeat("я", 5001))}},
		{"negative price", models.CreateServiceRequest{Name: "Сайт", Price: ptr(-1)}},
		{"negative position", models.CreateServiceRequest{Name: "Сайт", Position: -1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeServiceRepository()
			svc := services.NewServiceService(repo, testLogger())

			if _, err := svc.Create(tt.req); !errors.Is(err, services.ErrInvalidService) {
				t.Fatalf("expected ErrInvalidService, got %v", err)
			}
			if len(repo.services) != 0 {
				t.Error("invalid service must not be saved")
			}
		})
	}
}

func TestServiceService_Update_PatchesOnlyProvidedFields(t *testing.T) {
	original := newTestService(1, true)
	svc := services.NewServiceService(newFakeServiceRepository(original), testLogger())

	updated, err := svc.Update(1, models.UpdateServiceRequest{Price: ptr(60000)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.Price == nil || *updated.Price != 60000 {
		t.Errorf("price not updated: %v", updated.Price)
	}
	if updated.Name != original.Name || *updated.Description != *original.Description ||
		updated.Position != original.Position || updated.IsPublished != original.IsPublished {
		t.Errorf("unchanged fields were modified: %+v", updated)
	}
}

func TestServiceService_Update_InvalidMergedService(t *testing.T) {
	repo := newFakeServiceRepository(newTestService(1, true))
	svc := services.NewServiceService(repo, testLogger())

	tests := []struct {
		name string
		req  models.UpdateServiceRequest
	}{
		{"empty name", models.UpdateServiceRequest{Name: ptr("")}},
		{"negative price", models.UpdateServiceRequest{Price: ptr(-100)}},
		{"negative position", models.UpdateServiceRequest{Position: ptr(-1)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.Update(1, tt.req); !errors.Is(err, services.ErrInvalidService) {
				t.Fatalf("expected ErrInvalidService, got %v", err)
			}
			stored := repo.services[1]
			if stored.Name != "Лендинг" || *stored.Price != 50000 || stored.Position != 1 {
				t.Errorf("invalid update must not be saved: %+v", stored)
			}
		})
	}
}

func TestServiceService_Update_NotFound(t *testing.T) {
	svc := services.NewServiceService(newFakeServiceRepository(), testLogger())

	if _, err := svc.Update(99, models.UpdateServiceRequest{Name: ptr("Сайт")}); !errors.Is(err, repository.ErrServiceNotFound) {
		t.Fatalf("expected ErrServiceNotFound, got %v", err)
	}
}

func TestServiceService_Delete(t *testing.T) {
	repo := newFakeServiceRepository(newTestService(1, true))
	svc := services.NewServiceService(repo, testLogger())

	if err := svc.Delete(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := repo.services[1]; ok {
		t.Error("service was not deleted")
	}
}

func TestServiceService_Delete_NotFound(t *testing.T) {
	svc := services.NewServiceService(newFakeServiceRepository(), testLogger())

	if err := svc.Delete(99); !errors.Is(err, repository.ErrServiceNotFound) {
		t.Fatalf("expected ErrServiceNotFound, got %v", err)
	}
}

func TestServiceService_List_DoesNotForcePublished(t *testing.T) {
	svc := services.NewServiceService(newFakeServiceRepository(newTestService(1, true), newTestService(2, false)), testLogger())

	got, err := svc.List(services.ListAdminServicesFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("admin List must return published and unpublished, got %d items", len(got))
	}
}

func TestServiceService_List_FiltersUnpublished(t *testing.T) {
	svc := services.NewServiceService(newFakeServiceRepository(newTestService(1, true), newTestService(2, false)), testLogger())

	got, err := svc.List(services.ListAdminServicesFilter{IsPublished: ptr(false)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != 2 {
		t.Errorf("expected only unpublished #2, got %+v", got)
	}
}

func TestServiceService_GetByID_ReturnsUnpublished(t *testing.T) {
	svc := services.NewServiceService(newFakeServiceRepository(newTestService(1, false)), testLogger())

	got, err := svc.GetByID(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.IsPublished {
		t.Error("expected the unpublished item")
	}
}

func TestServiceService_GetByID_NotFound(t *testing.T) {
	svc := services.NewServiceService(newFakeServiceRepository(), testLogger())

	if _, err := svc.GetByID(99); !errors.Is(err, repository.ErrServiceNotFound) {
		t.Fatalf("expected ErrServiceNotFound, got %v", err)
	}
}
