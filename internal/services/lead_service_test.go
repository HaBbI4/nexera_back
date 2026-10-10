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

// fakeLeadRepository — реализация repository.LeadRepository в памяти. Запоминает последний
// переданный фильтр и созданные заявки; serviceIDs имитирует внешний ключ
// leads.service_id -> services.id.
type fakeLeadRepository struct {
	leads      map[int]*models.Lead
	serviceIDs map[int]bool
	lastFilter repository.LeadFilter
}

var _ repository.LeadRepository = (*fakeLeadRepository)(nil)

func newFakeLeadRepository(leads ...*models.Lead) *fakeLeadRepository {
	repo := &fakeLeadRepository{
		leads:      make(map[int]*models.Lead),
		serviceIDs: map[int]bool{1: true},
	}
	for _, l := range leads {
		repo.leads[l.ID] = l
	}
	return repo
}

func (f *fakeLeadRepository) List(filter repository.LeadFilter) ([]*models.Lead, error) {
	f.lastFilter = filter

	var result []*models.Lead
	for _, l := range f.leads {
		result = append(result, l)
	}
	return result, nil
}

func (f *fakeLeadRepository) GetByID(id int) (*models.Lead, error) {
	lead, ok := f.leads[id]
	if !ok {
		return nil, repository.ErrLeadNotFound
	}
	return lead, nil
}

func (f *fakeLeadRepository) Create(lead *models.Lead) error {
	if lead.ServiceID != nil && !f.serviceIDs[*lead.ServiceID] {
		return repository.ErrServiceNotFound
	}
	lead.ID = len(f.leads) + 1
	now := time.Now()
	lead.CreatedAt = now
	lead.UpdatedAt = now
	f.leads[lead.ID] = lead
	return nil
}

func (f *fakeLeadRepository) Delete(id int) error {
	if _, ok := f.leads[id]; !ok {
		return repository.ErrLeadNotFound
	}
	delete(f.leads, id)
	return nil
}

func validLeadRequest() models.CreateLeadRequest {
	return models.CreateLeadRequest{
		Name:  "Иван",
		Phone: "+79991234567",
	}
}

func TestLeadService_Create(t *testing.T) {
	repo := newFakeLeadRepository()
	svc := services.NewLeadService(repo, testLogger())

	req := validLeadRequest()
	req.ServiceID = ptr(1)
	req.Email = ptr("ivan@example.com")
	req.Message = ptr("Нужен сайт")
	req.Source = "landing"

	lead, err := svc.Create(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lead.ID == 0 {
		t.Error("lead ID not assigned")
	}
	if lead.Source != "landing" {
		t.Errorf("got source %q, want %q", lead.Source, "landing")
	}
	if _, ok := repo.leads[lead.ID]; !ok {
		t.Error("lead not saved")
	}
}

func TestLeadService_Create_Normalizes(t *testing.T) {
	svc := services.NewLeadService(newFakeLeadRepository(), testLogger())

	lead, err := svc.Create(models.CreateLeadRequest{
		Name:    "  Иван  ",
		Phone:   " +7 (999) 123-45-67 ",
		Email:   ptr("   "),
		Message: ptr(""),
		Source:  "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if lead.Name != "Иван" {
		t.Errorf("name not trimmed: %q", lead.Name)
	}
	if lead.Phone != "+79991234567" {
		t.Errorf("phone not normalized: %q", lead.Phone)
	}
	if lead.Email != nil || lead.Message != nil {
		t.Errorf("empty optional fields must become nil: email=%v message=%v", lead.Email, lead.Message)
	}
	if lead.Source != "contact_form" {
		t.Errorf("empty source must default to contact_form, got %q", lead.Source)
	}
}

func TestLeadService_Create_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(r *models.CreateLeadRequest)
	}{
		{"empty name", func(r *models.CreateLeadRequest) { r.Name = "   " }},
		{"name too long", func(r *models.CreateLeadRequest) { r.Name = strings.Repeat("я", 101) }},
		{"empty phone", func(r *models.CreateLeadRequest) { r.Phone = "" }},
		{"phone without plus", func(r *models.CreateLeadRequest) { r.Phone = "89991234567" }},
		{"phone with letters", func(r *models.CreateLeadRequest) { r.Phone = "+7999abc4567" }},
		{"phone too short", func(r *models.CreateLeadRequest) { r.Phone = "+712345" }},
		{"phone too long", func(r *models.CreateLeadRequest) { r.Phone = "+1234567890123456" }},
		{"malformed email", func(r *models.CreateLeadRequest) { r.Email = ptr("not-an-email") }},
		{"message too long", func(r *models.CreateLeadRequest) { r.Message = ptr(strings.Repeat("я", 5001)) }},
		{"source too long", func(r *models.CreateLeadRequest) { r.Source = strings.Repeat("a", 101) }},
		{"non-positive service_id", func(r *models.CreateLeadRequest) { r.ServiceID = ptr(0) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeLeadRepository()
			svc := services.NewLeadService(repo, testLogger())

			req := validLeadRequest()
			tt.mutate(&req)

			if _, err := svc.Create(req); !errors.Is(err, services.ErrInvalidLead) {
				t.Fatalf("expected ErrInvalidLead, got %v", err)
			}
			if len(repo.leads) != 0 {
				t.Error("invalid lead must not be saved")
			}
		})
	}
}

func TestLeadService_Create_ServiceNotFound(t *testing.T) {
	svc := services.NewLeadService(newFakeLeadRepository(), testLogger())

	req := validLeadRequest()
	req.ServiceID = ptr(99)

	if _, err := svc.Create(req); !errors.Is(err, repository.ErrServiceNotFound) {
		t.Fatalf("expected ErrServiceNotFound, got %v", err)
	}
}

func TestLeadService_List_PassesFilterToRepository(t *testing.T) {
	repo := newFakeLeadRepository()
	svc := services.NewLeadService(repo, testLogger())

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	filter := services.ListLeadsFilter{
		ServiceID:   ptr(1),
		Source:      ptr("landing"),
		CreatedFrom: &from,
		CreatedTo:   &to,
		Sort:        repository.LeadSortName,
		Order:       repository.SortAsc,
		Limit:       10,
		Offset:      20,
	}
	if _, err := svc.List(filter); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := repo.lastFilter
	if got.ServiceID == nil || *got.ServiceID != 1 || got.Source == nil || *got.Source != "landing" ||
		got.CreatedFrom == nil || !got.CreatedFrom.Equal(from) || got.CreatedTo == nil || !got.CreatedTo.Equal(to) ||
		got.Sort != repository.LeadSortName || got.Order != repository.SortAsc ||
		got.Limit != 10 || got.Offset != 20 {
		t.Errorf("filter not passed through correctly: %+v", got)
	}
}

func TestLeadService_GetByID_NotFound(t *testing.T) {
	svc := services.NewLeadService(newFakeLeadRepository(), testLogger())

	if _, err := svc.GetByID(99); !errors.Is(err, repository.ErrLeadNotFound) {
		t.Fatalf("expected ErrLeadNotFound, got %v", err)
	}
}

func TestLeadService_Delete(t *testing.T) {
	repo := newFakeLeadRepository(&models.Lead{ID: 1, Name: "Иван", Phone: "+79991234567"})
	svc := services.NewLeadService(repo, testLogger())

	if err := svc.Delete(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := repo.leads[1]; ok {
		t.Error("lead was not deleted")
	}
}

func TestLeadService_Delete_NotFound(t *testing.T) {
	svc := services.NewLeadService(newFakeLeadRepository(), testLogger())

	if err := svc.Delete(99); !errors.Is(err, repository.ErrLeadNotFound) {
		t.Fatalf("expected ErrLeadNotFound, got %v", err)
	}
}
