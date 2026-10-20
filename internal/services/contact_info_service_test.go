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

type fakeContactInfoRepository struct {
	items map[int]*models.ContactInfo
}

var _ repository.ContactInfoRepository = (*fakeContactInfoRepository)(nil)

func newFakeContactInfoRepository() *fakeContactInfoRepository {
	return &fakeContactInfoRepository{items: make(map[int]*models.ContactInfo)}
}

func (f *fakeContactInfoRepository) List(filter repository.ContactInfoFilter) ([]*models.ContactInfo, error) {
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
	return &copied, nil
}

func (f *fakeContactInfoRepository) Create(info *models.ContactInfo) error {
	info.ID = len(f.items) + 1
	now := time.Now()
	info.CreatedAt = now
	info.UpdatedAt = now
	stored := *info
	f.items[info.ID] = &stored
	return nil
}

func (f *fakeContactInfoRepository) Update(info *models.ContactInfo) error {
	if _, ok := f.items[info.ID]; !ok {
		return repository.ErrContactInfoNotFound
	}
	info.UpdatedAt = time.Now()
	stored := *info
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

func TestContactInfoService_Create_Invalid(t *testing.T) {
	valid := models.CreateContactInfoRequest{
		Type:  models.ContactInfoTypePhone,
		Value: "+7 900 000-00-00",
	}

	tests := []struct {
		name   string
		mutate func(r *models.CreateContactInfoRequest)
	}{
		{"invalid type", func(r *models.CreateContactInfoRequest) { r.Type = models.ContactInfoType("fax") }},
		{"empty type", func(r *models.CreateContactInfoRequest) { r.Type = models.ContactInfoType("") }},
		{"empty value", func(r *models.CreateContactInfoRequest) { r.Value = "" }},
		{"value too long", func(r *models.CreateContactInfoRequest) { r.Value = strings.Repeat("я", 101) }},
		{"label too long", func(r *models.CreateContactInfoRequest) { r.Label = ptr(strings.Repeat("я", 101)) }},
		{"negative position", func(r *models.CreateContactInfoRequest) { r.Position = -1 }},
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

func TestContactInfoService_Create_Valid(t *testing.T) {
	repo := newFakeContactInfoRepository()
	svc := services.NewContactInfoService(repo, testLogger())

	info, err := svc.Create(models.CreateContactInfoRequest{
		Type:        models.ContactInfoTypeEmail,
		Value:       "hello@example.com",
		Position:    1,
		IsPublished: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ID == 0 {
		t.Error("contact info ID not assigned")
	}
	if _, ok := repo.items[info.ID]; !ok {
		t.Error("contact info not saved")
	}
}
