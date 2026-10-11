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

// fakeRoleRepository — реализация repository.RoleRepository в памяти. GetByID возвращает
// копию, чтобы изменения сервиса не "сохранялись" без вызова Update. inUse имитирует
// ON DELETE RESTRICT от users.role_id: такие роли Delete не удаляет.
type fakeRoleRepository struct {
	roles map[int]*models.Role
	inUse map[int]bool
}

var _ repository.RoleRepository = (*fakeRoleRepository)(nil)

func newFakeRoleRepository(roles ...*models.Role) *fakeRoleRepository {
	repo := &fakeRoleRepository{
		roles: make(map[int]*models.Role),
		inUse: make(map[int]bool),
	}
	for _, r := range roles {
		copied := *r
		repo.roles[r.ID] = &copied
	}
	return repo
}

func (f *fakeRoleRepository) List() ([]*models.Role, error) {
	var result []*models.Role
	for _, r := range f.roles {
		result = append(result, r)
	}
	return result, nil
}

func (f *fakeRoleRepository) GetByID(id int) (*models.Role, error) {
	role, ok := f.roles[id]
	if !ok {
		return nil, repository.ErrRoleNotFound
	}
	copied := *role
	return &copied, nil
}

func (f *fakeRoleRepository) GetByCode(code string) (*models.Role, error) {
	for _, r := range f.roles {
		if r.Code == code {
			copied := *r
			return &copied, nil
		}
	}
	return nil, repository.ErrRoleNotFound
}

func (f *fakeRoleRepository) codeTaken(code string, exceptID int) bool {
	for _, r := range f.roles {
		if r.ID != exceptID && r.Code == code {
			return true
		}
	}
	return false
}

func (f *fakeRoleRepository) Create(role *models.Role) error {
	if f.codeTaken(role.Code, 0) {
		return repository.ErrRoleCodeTaken
	}
	role.ID = len(f.roles) + 1
	now := time.Now()
	role.CreatedAt = now
	role.UpdatedAt = now
	stored := *role
	f.roles[role.ID] = &stored
	return nil
}

func (f *fakeRoleRepository) Update(role *models.Role) error {
	if _, ok := f.roles[role.ID]; !ok {
		return repository.ErrRoleNotFound
	}
	if f.codeTaken(role.Code, role.ID) {
		return repository.ErrRoleCodeTaken
	}
	role.UpdatedAt = time.Now()
	stored := *role
	f.roles[role.ID] = &stored
	return nil
}

func (f *fakeRoleRepository) Delete(id int) error {
	if _, ok := f.roles[id]; !ok {
		return repository.ErrRoleNotFound
	}
	if f.inUse[id] {
		return repository.ErrRoleInUse
	}
	delete(f.roles, id)
	return nil
}

func TestRoleService_Create(t *testing.T) {
	repo := newFakeRoleRepository()
	svc := services.NewRoleService(repo, testLogger())

	role, err := svc.Create(models.CreateRoleRequest{Name: "Контент-менеджер", Code: "content_manager"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role.ID == 0 {
		t.Error("role ID not assigned")
	}
	if _, ok := repo.roles[role.ID]; !ok {
		t.Error("role not saved")
	}
}

func TestRoleService_Create_Invalid(t *testing.T) {
	tests := []struct {
		name string
		req  models.CreateRoleRequest
	}{
		{"empty name", models.CreateRoleRequest{Name: "", Code: "manager"}},
		{"name too long", models.CreateRoleRequest{Name: strings.Repeat("я", 101), Code: "manager"}},
		{"empty code", models.CreateRoleRequest{Name: "Менеджер", Code: ""}},
		{"uppercase code", models.CreateRoleRequest{Name: "Менеджер", Code: "Manager"}},
		{"code starts with digit", models.CreateRoleRequest{Name: "Менеджер", Code: "1manager"}},
		{"code too short", models.CreateRoleRequest{Name: "Менеджер", Code: "m"}},
		{"code with dash", models.CreateRoleRequest{Name: "Менеджер", Code: "content-manager"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRoleRepository()
			svc := services.NewRoleService(repo, testLogger())

			if _, err := svc.Create(tt.req); !errors.Is(err, services.ErrInvalidRole) {
				t.Fatalf("expected ErrInvalidRole, got %v", err)
			}
			if len(repo.roles) != 0 {
				t.Error("invalid role must not be saved")
			}
		})
	}
}

func TestRoleService_Create_CodeTaken(t *testing.T) {
	svc := services.NewRoleService(newFakeRoleRepository(testManagerRole), testLogger())

	_, err := svc.Create(models.CreateRoleRequest{Name: "Еще менеджер", Code: testManagerRole.Code})
	if !errors.Is(err, repository.ErrRoleCodeTaken) {
		t.Fatalf("expected ErrRoleCodeTaken, got %v", err)
	}
}

func TestRoleService_Update_PatchesOnlyProvidedFields(t *testing.T) {
	svc := services.NewRoleService(newFakeRoleRepository(testManagerRole), testLogger())

	updated, err := svc.Update(testManagerRole.ID, models.UpdateRoleRequest{Name: ptr("Старший менеджер")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "Старший менеджер" {
		t.Errorf("name not updated: %q", updated.Name)
	}
	if updated.Code != testManagerRole.Code {
		t.Errorf("code must stay the same, got %q", updated.Code)
	}
}

func TestRoleService_Update_AdminCodeIsImmutable(t *testing.T) {
	repo := newFakeRoleRepository(testAdminRole)
	svc := services.NewRoleService(repo, testLogger())

	_, err := svc.Update(testAdminRole.ID, models.UpdateRoleRequest{Code: ptr("superuser")})
	if !errors.Is(err, services.ErrInvalidRole) {
		t.Fatalf("expected ErrInvalidRole, got %v", err)
	}
	if repo.roles[testAdminRole.ID].Code != models.RoleCodeAdmin {
		t.Error("admin role code must not be changed")
	}
}

func TestRoleService_Update_AdminNameCanChange(t *testing.T) {
	svc := services.NewRoleService(newFakeRoleRepository(testAdminRole), testLogger())

	// Тот же code в запросе — не попытка его поменять.
	updated, err := svc.Update(testAdminRole.ID, models.UpdateRoleRequest{
		Name: ptr("Суперадмин"),
		Code: ptr(models.RoleCodeAdmin),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "Суперадмин" {
		t.Errorf("name not updated: %q", updated.Name)
	}
}

func TestRoleService_Update_Invalid(t *testing.T) {
	repo := newFakeRoleRepository(testManagerRole)
	svc := services.NewRoleService(repo, testLogger())

	_, err := svc.Update(testManagerRole.ID, models.UpdateRoleRequest{Code: ptr("Bad Code")})
	if !errors.Is(err, services.ErrInvalidRole) {
		t.Fatalf("expected ErrInvalidRole, got %v", err)
	}
	if repo.roles[testManagerRole.ID].Code != testManagerRole.Code {
		t.Error("invalid update must not be saved")
	}
}

func TestRoleService_Update_NotFound(t *testing.T) {
	svc := services.NewRoleService(newFakeRoleRepository(), testLogger())

	_, err := svc.Update(99, models.UpdateRoleRequest{Name: ptr("Кто-то")})
	if !errors.Is(err, repository.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestRoleService_Delete(t *testing.T) {
	repo := newFakeRoleRepository(testManagerRole)
	svc := services.NewRoleService(repo, testLogger())

	if err := svc.Delete(testManagerRole.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := repo.roles[testManagerRole.ID]; ok {
		t.Error("role was not deleted")
	}
}

func TestRoleService_Delete_InUse(t *testing.T) {
	repo := newFakeRoleRepository(testManagerRole)
	repo.inUse[testManagerRole.ID] = true
	svc := services.NewRoleService(repo, testLogger())

	if err := svc.Delete(testManagerRole.ID); !errors.Is(err, repository.ErrRoleInUse) {
		t.Fatalf("expected ErrRoleInUse, got %v", err)
	}
}
