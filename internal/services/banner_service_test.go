package services_test

import (
	"errors"
	"testing"
	"time"

	"nexera_back/internal/models"
	"nexera_back/internal/repository"
	"nexera_back/internal/services"
)

// fakeBannerRepository — реализация repository.BannerRepository в памяти.
// Запоминает последний переданный фильтр, чтобы тесты могли проверить,
// что сервис действительно применяет бизнес-правило видимости.
type fakeBannerRepository struct {
	banners    map[int]*models.Banner
	lastFilter repository.BannerFilter
}

var _ repository.BannerRepository = (*fakeBannerRepository)(nil)

func newFakeBannerRepository(banners ...*models.Banner) *fakeBannerRepository {
	repo := &fakeBannerRepository{banners: make(map[int]*models.Banner)}
	for _, b := range banners {
		repo.banners[b.ID] = b
	}
	return repo
}

func (f *fakeBannerRepository) List(filter repository.BannerFilter) ([]*models.Banner, error) {
	f.lastFilter = filter

	var result []*models.Banner
	for _, b := range f.banners {
		if filter.PageID != nil && b.PageID != *filter.PageID {
			continue
		}
		if filter.OnlyVisible && !isVisibleForTest(b, filter.ActiveAt) {
			continue
		}
		result = append(result, b)
	}
	return result, nil
}

func (f *fakeBannerRepository) GetByID(id int) (*models.Banner, error) {
	banner, ok := f.banners[id]
	if !ok {
		return nil, repository.ErrBannerNotFound
	}
	return banner, nil
}

func (f *fakeBannerRepository) Create(banner *models.Banner) error {
	banner.ID = len(f.banners) + 1
	now := time.Now()
	banner.CreatedAt = now
	banner.UpdatedAt = now
	f.banners[banner.ID] = banner
	return nil
}

func (f *fakeBannerRepository) Update(banner *models.Banner) error {
	if _, ok := f.banners[banner.ID]; !ok {
		return repository.ErrBannerNotFound
	}
	banner.UpdatedAt = time.Now()
	f.banners[banner.ID] = banner
	return nil
}

func (f *fakeBannerRepository) Delete(id int) error {
	if _, ok := f.banners[id]; !ok {
		return repository.ErrBannerNotFound
	}
	delete(f.banners, id)
	return nil
}

func isVisibleForTest(b *models.Banner, now time.Time) bool {
	if !b.IsActive {
		return false
	}
	if b.StartsAt != nil && b.StartsAt.After(now) {
		return false
	}
	if b.EndsAt != nil && b.EndsAt.Before(now) {
		return false
	}
	return true
}

func TestBannerService_ListPublic_AppliesVisibilityRule(t *testing.T) {
	active := &models.Banner{ID: 1, PageID: 10, IsActive: true}
	repo := newFakeBannerRepository(active)
	svc := services.NewBannerService(repo, testLogger())

	if _, err := svc.ListPublic(services.ListBannersFilter{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !repo.lastFilter.OnlyVisible {
		t.Error("expected ListPublic to always request OnlyVisible banners")
	}
	if repo.lastFilter.ActiveAt.IsZero() {
		t.Error("expected ListPublic to pass a non-zero ActiveAt")
	}
}

func TestBannerService_ListPublic_FiltersByPage(t *testing.T) {
	bannerPage1 := &models.Banner{ID: 1, PageID: 1, IsActive: true}
	bannerPage2 := &models.Banner{ID: 2, PageID: 2, IsActive: true}
	repo := newFakeBannerRepository(bannerPage1, bannerPage2)
	svc := services.NewBannerService(repo, testLogger())

	pageID := 1
	got, err := svc.ListPublic(services.ListBannersFilter{PageID: &pageID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != bannerPage1.ID {
		t.Errorf("expected only banner from page 1, got %+v", got)
	}
}

func TestBannerService_GetPublicByID_ActiveBanner(t *testing.T) {
	active := &models.Banner{ID: 1, IsActive: true}
	svc := services.NewBannerService(newFakeBannerRepository(active), testLogger())

	got, err := svc.GetPublicByID(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 1 {
		t.Errorf("got ID %d, want 1", got.ID)
	}
}

func TestBannerService_GetPublicByID_InactiveBannerIsNotFound(t *testing.T) {
	inactive := &models.Banner{ID: 1, IsActive: false}
	svc := services.NewBannerService(newFakeBannerRepository(inactive), testLogger())

	if _, err := svc.GetPublicByID(1); err == nil {
		t.Fatal("expected error for inactive banner, got nil")
	}
}

func TestBannerService_GetPublicByID_NotYetStartedBannerIsNotFound(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	notYetStarted := &models.Banner{ID: 1, IsActive: true, StartsAt: &future}
	svc := services.NewBannerService(newFakeBannerRepository(notYetStarted), testLogger())

	if _, err := svc.GetPublicByID(1); err == nil {
		t.Fatal("expected error for not-yet-started banner, got nil")
	}
}

func TestBannerService_GetPublicByID_ExpiredBannerIsNotFound(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour)
	expired := &models.Banner{ID: 1, IsActive: true, EndsAt: &past}
	svc := services.NewBannerService(newFakeBannerRepository(expired), testLogger())

	if _, err := svc.GetPublicByID(1); err == nil {
		t.Fatal("expected error for expired banner, got nil")
	}
}

func TestBannerService_GetPublicByID_Missing(t *testing.T) {
	svc := services.NewBannerService(newFakeBannerRepository(), testLogger())

	if _, err := svc.GetPublicByID(999); err == nil {
		t.Fatal("expected error for missing banner, got nil")
	}
}

func TestBannerService_Create_Success(t *testing.T) {
	svc := services.NewBannerService(newFakeBannerRepository(), testLogger())

	req := models.CreateBannerRequest{
		PageID:   1,
		Title:    "Main banner",
		ImageURL: "https://example.com/banner.png",
	}

	banner, err := svc.Create(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if banner.ID == 0 {
		t.Error("expected banner to be assigned an ID")
	}
	if banner.CreatedAt.IsZero() || banner.UpdatedAt.IsZero() {
		t.Error("expected CreatedAt/UpdatedAt to be set")
	}
}

func TestBannerService_Create_MissingTitleIsInvalid(t *testing.T) {
	svc := services.NewBannerService(newFakeBannerRepository(), testLogger())

	req := models.CreateBannerRequest{PageID: 1, ImageURL: "https://example.com/banner.png"}

	_, err := svc.Create(req)
	if !errors.Is(err, services.ErrInvalidBanner) {
		t.Fatalf("expected ErrInvalidBanner, got %v", err)
	}
}

func TestBannerService_Create_EndsAtBeforeStartsAtIsInvalid(t *testing.T) {
	svc := services.NewBannerService(newFakeBannerRepository(), testLogger())

	starts := time.Now().Add(24 * time.Hour)
	ends := time.Now()
	req := models.CreateBannerRequest{
		PageID:   1,
		Title:    "Main banner",
		ImageURL: "https://example.com/banner.png",
		StartsAt: &starts,
		EndsAt:   &ends,
	}

	_, err := svc.Create(req)
	if !errors.Is(err, services.ErrInvalidBanner) {
		t.Fatalf("expected ErrInvalidBanner, got %v", err)
	}
}

func TestBannerService_Update_PartialMergeLeavesOtherFieldsUnchanged(t *testing.T) {
	existing := &models.Banner{ID: 1, PageID: 1, Title: "Old title", ImageURL: "https://example.com/old.png", Position: 5}
	svc := services.NewBannerService(newFakeBannerRepository(existing), testLogger())

	newTitle := "New title"
	got, err := svc.Update(1, models.UpdateBannerRequest{Title: &newTitle})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != newTitle {
		t.Errorf("expected title to be updated, got %q", got.Title)
	}
	if got.ImageURL != "https://example.com/old.png" || got.Position != 5 {
		t.Errorf("expected untouched fields to remain unchanged, got %+v", got)
	}
}

func TestBannerService_Update_NotFound(t *testing.T) {
	svc := services.NewBannerService(newFakeBannerRepository(), testLogger())

	newTitle := "New title"
	_, err := svc.Update(999, models.UpdateBannerRequest{Title: &newTitle})
	if !errors.Is(err, repository.ErrBannerNotFound) {
		t.Fatalf("expected ErrBannerNotFound, got %v", err)
	}
}

func TestBannerService_Update_ResultingStateMustStillBeValid(t *testing.T) {
	existing := &models.Banner{ID: 1, PageID: 1, Title: "Old title", ImageURL: "https://example.com/old.png"}
	svc := services.NewBannerService(newFakeBannerRepository(existing), testLogger())

	emptyTitle := ""
	_, err := svc.Update(1, models.UpdateBannerRequest{Title: &emptyTitle})
	if !errors.Is(err, services.ErrInvalidBanner) {
		t.Fatalf("expected ErrInvalidBanner, got %v", err)
	}
}

func TestBannerService_Delete_Success(t *testing.T) {
	existing := &models.Banner{ID: 1, PageID: 1, Title: "Banner", ImageURL: "https://example.com/b.png"}
	svc := services.NewBannerService(newFakeBannerRepository(existing), testLogger())

	if err := svc.Delete(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := svc.GetPublicByID(1); !errors.Is(err, repository.ErrBannerNotFound) {
		t.Fatalf("expected banner to be gone after delete, got %v", err)
	}
}

func TestBannerService_Delete_NotFound(t *testing.T) {
	svc := services.NewBannerService(newFakeBannerRepository(), testLogger())

	if err := svc.Delete(999); !errors.Is(err, repository.ErrBannerNotFound) {
		t.Fatalf("expected ErrBannerNotFound, got %v", err)
	}
}
