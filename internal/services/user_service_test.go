package services_test

import (
	"fmt"
	"testing"

	"nexera_back/internal/config"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/repository"
	"nexera_back/internal/services"
)

// fakeUserRepository — реализация repository.UserRepository в памяти, без реальной БД.
// Это и есть практический смысл repository-слоя: сервис ничего не знает про Postgres
// и его можно протестировать, просто подставив сюда другую реализацию интерфейса.
type fakeUserRepository struct {
	byEmail      map[string]*models.User
	byID         map[int]*models.User
	lastLoginIDs []int
}

var _ repository.UserRepository = (*fakeUserRepository)(nil)

func newFakeUserRepository(users ...*models.User) *fakeUserRepository {
	repo := &fakeUserRepository{
		byEmail: make(map[string]*models.User),
		byID:    make(map[int]*models.User),
	}
	for _, u := range users {
		repo.byEmail[u.Email] = u
		repo.byID[u.ID] = u
	}
	return repo
}

func (f *fakeUserRepository) GetByEmail(email string) (*models.User, error) {
	user, ok := f.byEmail[email]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

func (f *fakeUserRepository) GetByID(id int) (*models.User, error) {
	user, ok := f.byID[id]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

func (f *fakeUserRepository) UpdateLastLogin(id int) error {
	f.lastLoginIDs = append(f.lastLoginIDs, id)
	return nil
}

func testLogger() *logger.Logger {
	return logger.New(&config.LoggerConfig{Level: "error", Format: "text"})
}

func TestUserService_GetByEmail(t *testing.T) {
	admin := &models.User{ID: 1, Email: "admin@nexera.test", IsActive: true, Role: &models.Role{Code: "admin"}}
	svc := services.NewUserService(newFakeUserRepository(admin), testLogger())

	got, err := svc.GetByEmail("admin@nexera.test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != admin.ID {
		t.Errorf("got ID %d, want %d", got.ID, admin.ID)
	}
}

func TestUserService_GetByEmail_NotFound(t *testing.T) {
	svc := services.NewUserService(newFakeUserRepository(), testLogger())

	if _, err := svc.GetByEmail("missing@nexera.test"); err == nil {
		t.Fatal("expected error for missing user, got nil")
	}
}

func TestUserService_GetByID(t *testing.T) {
	admin := &models.User{ID: 42, Email: "admin@nexera.test"}
	svc := services.NewUserService(newFakeUserRepository(admin), testLogger())

	got, err := svc.GetByID(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Email != admin.Email {
		t.Errorf("got email %q, want %q", got.Email, admin.Email)
	}
}

func TestUserService_UpdateLastLogin(t *testing.T) {
	admin := &models.User{ID: 1, Email: "admin@nexera.test"}
	repo := newFakeUserRepository(admin)
	svc := services.NewUserService(repo, testLogger())

	if err := svc.UpdateLastLogin(admin.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.lastLoginIDs) != 1 || repo.lastLoginIDs[0] != admin.ID {
		t.Errorf("UpdateLastLogin not recorded correctly: %v", repo.lastLoginIDs)
	}
}
