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

// fakeContactInfoRepository — реализация repository.ContactInfoRepository в памяти.
// Запоминает последний фильтр List, чтобы тесты проверяли бизнес-правило видимости.
type fakeContactInfoRepository struct {
	items      map[int]*models.ContactInfo
	lastFilter repository.ContactInfoFilter
}

var _ repository.ContactInfoRepository = (*fakeContactInfoRepository)(nil)

func newFakeContactInfoRepository(items ...*models.ContactInfo) *fakeContactInfoRepository {
	repo := &fakeContactInfoRepository{items: make(map[int]*models.ContactInfo)}
	for _, item := range items {
		copied := *item
		if item.Value != nil {
			v := *item.Value
			copied.Value = &v
		}
		if item.Label != nil {
			l := *item.Label
			copied.Label = &l
		}
		repo.items[item.ID] = &copied
	}
	return repo
}

func (f *fakeContactInfoRepository) List(filter repository.ContactInfoFilter) ([]*models.ContactInfo, error) {
	f.lastFilter = filter
	var result []*models.ContactInfo
	for _, item := range f.items {
		if filter.IsPublished != nil && item.IsPublished != *filter.IsPublished {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

func (f *fakeContactInfoRepository) GetByID(id int) (*models.ContactInfo, error) {
	item, ok := f.items[id]
	if !ok {
		return nil, repository.ErrContactInfoNotFound
	}
	copied := *item
	if item.Value != nil {
		v := *item.Value
		copied.Value = &v
	}
	if item.Label != nil {
		l := *item.Label
		copied.Label = &l
	}
	return &copied, nil
}

func (f *fakeContactInfoRepository) Create(info *models.ContactInfo) error {
	info.ID = len(f.items) + 1
	now := time.Now()
	info.CreatedAt = now
	info.UpdatedAt = now
	stored := *info
	if info.Value != nil {
		v := *info.Value
		stored.Value = &v
	}
	if info.Label != nil {
		l := *info.Label
		stored.Label = &l
	}
	f.items[info.ID] = &stored
	return nil
}

func (f *fakeContactInfoRepository) Update(info *models.ContactInfo) error {
	if _, ok := f.items[info.ID]; !ok {
		return repository.ErrContactInfoNotFound
	}
	info.UpdatedAt = time.Now()
	stored := *info
	if info.Value != nil {
		v := *info.Value
		stored.Value = &v
	}
	if info.Label != nil {
		l := *info.Label
		stored.Label = &l
	} else {
		stored.Label = nil
	}
	f.items[info.ID] = &stored
	return nil
}

func (f *fakeContactInfoRepository) Delete(id int) error {
	if _, ok := f.items[id]; !ok {
		return repository.ErrContactInfoNotFound
	}
	delete(f.items, id)
	return nil
}

func newTestContactInfo(id int, published bool) *models.ContactInfo {
	value := "hello@example.com"
	return &models.ContactInfo{
		ID:          id,
		Type:        models.ContactInfoTypeEmail,
		Value:       &value,
		Position:    id,
		IsPublished: published,
	}
}

func TestContactInfoService_ListPublic_ForcesOnlyPublished(t *testing.T) {
	repo := newFakeContactInfoRepository(newTestContactInfo(1, true), newTestContactInfo(2, false))
	svc := services.NewContactInfoService(repo, testLogger())

	got, err := svc.ListPublic(services.ListContactInfoFilter{
		Sort:   repository.ContactInfoSortCreatedAt,
		Order:  repository.SortDesc,
		Limit:  10,
		Offset: 5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != 1 {
		t.Errorf("expected only published #1, got %+v", got)
	}

	f := repo.lastFilter
	if f.IsPublished == nil || !*f.IsPublished || f.Sort != repository.ContactInfoSortCreatedAt ||
		f.Order != repository.SortDesc || f.Limit != 10 || f.Offset != 5 {
		t.Errorf("filter not passed through correctly: %+v", f)
	}
}

func TestContactInfoService_GetPublicByID(t *testing.T) {
	svc := services.NewContactInfoService(newFakeContactInfoRepository(newTestContactInfo(1, true)), testLogger())

	got, err := svc.GetPublicByID(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 1 {
		t.Errorf("got ID %d, want 1", got.ID)
	}
}

func TestContactInfoService_GetPublicByID_UnpublishedIsNotFound(t *testing.T) {
	svc := services.NewContactInfoService(newFakeContactInfoRepository(newTestContactInfo(1, false)), testLogger())

	if _, err := svc.GetPublicByID(1); !errors.Is(err, repository.ErrContactInfoNotFound) {
		t.Fatalf("expected ErrContactInfoNotFound, got %v", err)
	}
}

func TestContactInfoService_GetPublicByID_NotFound(t *testing.T) {
	svc := services.NewContactInfoService(newFakeContactInfoRepository(), testLogger())

	if _, err := svc.GetPublicByID(99); !errors.Is(err, repository.ErrContactInfoNotFound) {
		t.Fatalf("expected ErrContactInfoNotFound, got %v", err)
	}
}

func TestContactInfoService_Create(t *testing.T) {
	repo := newFakeContactInfoRepository()
	svc := services.NewContactInfoService(repo, testLogger())

	info, err := svc.Create(models.CreateContactInfoRequest{
		Type:        models.ContactInfoTypeEmail,
		Value:       "hello@example.com",
		Label:       ptr("  "),
		Position:    3,
		IsPublished: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ID == 0 {
		t.Error("contact info ID not assigned")
	}
	if info.Label != nil {
		t.Errorf("empty label must become nil, got %q", *info.Label)
	}
	if _, ok := repo.items[info.ID]; !ok {
		t.Error("contact info not saved")
	}
}

func TestContactInfoService_Create_Invalid(t *testing.T) {
	valid := models.CreateContactInfoRequest{
		Type:  models.ContactInfoTypePhone,
		Value: "+7 (900) 000-00-00",
	}

	tests := []struct {
		name   string
		mutate func(r *models.CreateContactInfoRequest)
	}{
		{"invalid type", func(r *models.CreateContactInfoRequest) { r.Type = models.ContactInfoType("fax") }},
		{"empty type", func(r *models.CreateContactInfoRequest) { r.Type = models.ContactInfoType("") }},
		{"empty value", func(r *models.CreateContactInfoRequest) { r.Value = "" }},
		{"value too long", func(r *models.CreateContactInfoRequest) { r.Value = strings.Repeat("1", 101) }},
		{"label too long", func(r *models.CreateContactInfoRequest) { r.Label = ptr(strings.Repeat("я", 101)) }},
		{"negative position", func(r *models.CreateContactInfoRequest) { r.Position = -1 }},
		{"bad email", func(r *models.CreateContactInfoRequest) {
			r.Type = models.ContactInfoTypeEmail
			r.Value = "not-an-email"
		}},
		{"bad phone", func(r *models.CreateContactInfoRequest) {
			r.Type = models.ContactInfoTypePhone
			r.Value = "89001234567"
		}},
		{"bad telegram", func(r *models.CreateContactInfoRequest) {
			r.Type = models.ContactInfoTypeTelegram
			r.Value = "ab"
		}},
		{"bad vk url", func(r *models.CreateContactInfoRequest) {
			r.Type = models.ContactInfoTypeVK
			r.Value = "https://example.com/page"
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeContactInfoRepository()
			svc := services.NewContactInfoService(repo, testLogger())

			req := valid
			tt.mutate(&req)

			if _, err := svc.Create(req); !errors.Is(err, services.ErrInvalidContactInfo) {
				t.Fatalf("expected ErrInvalidContactInfo, got %v", err)
			}
			if len(repo.items) != 0 {
				t.Error("invalid contact info must not be saved")
			}
		})
	}
}

func TestContactInfoService_Create_NormalizesPhone(t *testing.T) {
	repo := newFakeContactInfoRepository()
	svc := services.NewContactInfoService(repo, testLogger())

	info, err := svc.Create(models.CreateContactInfoRequest{
		Type:  models.ContactInfoTypePhone,
		Value: "+7 (900) 123-45-67",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Value == nil || *info.Value != "+79001234567" {
		t.Fatalf("expected normalized phone, got %#v", info.Value)
	}
}

func TestContactInfoService_Update_PatchesOnlyProvidedFields(t *testing.T) {
	original := newTestContactInfo(1, true)
	label := "Офис"
	original.Label = &label
	svc := services.NewContactInfoService(newFakeContactInfoRepository(original), testLogger())

	updated, err := svc.Update(1, models.UpdateContactInfoRequest{IsPublished: ptr(false)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.IsPublished {
		t.Error("is_published not updated")
	}
	if updated.Type != original.Type || updated.Position != original.Position {
		t.Errorf("unchanged fields were modified: %+v", updated)
	}
	if updated.Label == nil || *updated.Label != "Офис" {
		t.Errorf("label should stay, got %#v", updated.Label)
	}
}

func TestContactInfoService_Update_ClearsEmptyLabel(t *testing.T) {
	original := newTestContactInfo(1, true)
	label := "Офис"
	original.Label = &label
	repo := newFakeContactInfoRepository(original)
	svc := services.NewContactInfoService(repo, testLogger())

	updated, err := svc.Update(1, models.UpdateContactInfoRequest{Label: ptr("")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Label != nil {
		t.Errorf("empty label must become nil, got %q", *updated.Label)
	}
	if repo.items[1].Label != nil {
		t.Error("stored label must be nil")
	}
}

func TestContactInfoService_Update_InvalidMerged(t *testing.T) {
	original := newTestContactInfo(1, true)
	repo := newFakeContactInfoRepository(original)
	svc := services.NewContactInfoService(repo, testLogger())

	tests := []struct {
		name string
		req  models.UpdateContactInfoRequest
	}{
		{"empty value", models.UpdateContactInfoRequest{Value: ptr("")}},
		{"negative position", models.UpdateContactInfoRequest{Position: ptr(-5)}},
		{"bad email", models.UpdateContactInfoRequest{Value: ptr("broken")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.Update(1, tt.req); !errors.Is(err, services.ErrInvalidContactInfo) {
				t.Fatalf("expected ErrInvalidContactInfo, got %v", err)
			}
			stored := repo.items[1]
			if stored.IsPublished != original.IsPublished || *stored.Value != *original.Value {
				t.Error("invalid update must not be saved")
			}
		})
	}
}

func TestContactInfoService_Update_NotFound(t *testing.T) {
	svc := services.NewContactInfoService(newFakeContactInfoRepository(), testLogger())

	if _, err := svc.Update(99, models.UpdateContactInfoRequest{Value: ptr("a@b.co")}); !errors.Is(err, repository.ErrContactInfoNotFound) {
		t.Fatalf("expected ErrContactInfoNotFound, got %v", err)
	}
}

func TestContactInfoService_Delete(t *testing.T) {
	repo := newFakeContactInfoRepository(newTestContactInfo(1, true))
	svc := services.NewContactInfoService(repo, testLogger())

	if err := svc.Delete(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := repo.items[1]; ok {
		t.Error("contact info was not deleted")
	}
}

func TestContactInfoService_Delete_NotFound(t *testing.T) {
	svc := services.NewContactInfoService(newFakeContactInfoRepository(), testLogger())

	if err := svc.Delete(99); !errors.Is(err, repository.ErrContactInfoNotFound) {
		t.Fatalf("expected ErrContactInfoNotFound, got %v", err)
	}
}

func TestContactInfoService_List_DoesNotForcePublished(t *testing.T) {
	svc := services.NewContactInfoService(
		newFakeContactInfoRepository(newTestContactInfo(1, true), newTestContactInfo(2, false)),
		testLogger(),
	)

	got, err := svc.List(services.ListAdminContactInfoFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("admin List must return published and unpublished, got %d items", len(got))
	}
}

func TestContactInfoService_List_FiltersUnpublished(t *testing.T) {
	svc := services.NewContactInfoService(
		newFakeContactInfoRepository(newTestContactInfo(1, true), newTestContactInfo(2, false)),
		testLogger(),
	)

	got, err := svc.List(services.ListAdminContactInfoFilter{IsPublished: ptr(false)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != 2 {
		t.Errorf("expected only unpublished #2, got %+v", got)
	}
}

func TestContactInfoService_GetByID_ReturnsUnpublished(t *testing.T) {
	svc := services.NewContactInfoService(newFakeContactInfoRepository(newTestContactInfo(1, false)), testLogger())

	got, err := svc.GetByID(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.IsPublished {
		t.Error("expected the unpublished item")
	}
}

func TestContactInfoService_GetByID_NotFound(t *testing.T) {
	svc := services.NewContactInfoService(newFakeContactInfoRepository(), testLogger())

	if _, err := svc.GetByID(99); !errors.Is(err, repository.ErrContactInfoNotFound) {
		t.Fatalf("expected ErrContactInfoNotFound, got %v", err)
	}
}
