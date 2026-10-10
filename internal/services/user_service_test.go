package services_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"nexera_back/internal/auth"
	"nexera_back/internal/config"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/repository"
	"nexera_back/internal/services"
)

// fakeUserRepository — реализация repository.UserRepository в памяти, без реальной БД.
// Это и есть практический смысл repository-слоя: сервис ничего не знает про Postgres
// и его можно протестировать, просто подставив сюда другую реализацию интерфейса.
//
// GetByID/GetByEmail возвращают копию — как и настоящая БД: изменения, которые сервис
// внес в сущность, не сохраняются, пока он не вызвал Update. roles имитирует внешний ключ
// users.role_id -> roles.id и заполнение user.Role через JOIN.
type fakeUserRepository struct {
	users        map[int]*models.User
	roles        map[int]*models.Role
	lastFilter   repository.UserFilter
	lastLoginIDs []int
}

var _ repository.UserRepository = (*fakeUserRepository)(nil)

func newFakeUserRepository(users ...*models.User) *fakeUserRepository {
	repo := &fakeUserRepository{
		users: make(map[int]*models.User),
		roles: make(map[int]*models.Role),
	}
	for _, u := range users {
		repo.users[u.ID] = u
		if u.Role != nil {
			repo.roles[u.RoleID] = u.Role
		}
	}
	return repo
}

func (f *fakeUserRepository) withRoles(roles ...*models.Role) *fakeUserRepository {
	for _, r := range roles {
		f.roles[r.ID] = r
	}
	return f
}

func (f *fakeUserRepository) List(filter repository.UserFilter) ([]*models.User, error) {
	f.lastFilter = filter

	var result []*models.User
	for _, u := range f.users {
		if filter.RoleID != nil && u.RoleID != *filter.RoleID {
			continue
		}
		if filter.IsActive != nil && u.IsActive != *filter.IsActive {
			continue
		}
		result = append(result, u)
	}
	return result, nil
}

func (f *fakeUserRepository) GetByEmail(email string) (*models.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			copied := *u
			return &copied, nil
		}
	}
	return nil, repository.ErrUserNotFound
}

func (f *fakeUserRepository) GetByID(id int) (*models.User, error) {
	user, ok := f.users[id]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	copied := *user
	return &copied, nil
}

func (f *fakeUserRepository) emailTaken(email string, exceptID int) bool {
	for _, u := range f.users {
		if u.ID != exceptID && strings.EqualFold(u.Email, email) {
			return true
		}
	}
	return false
}

func (f *fakeUserRepository) Create(user *models.User) error {
	if f.emailTaken(user.Email, 0) {
		return repository.ErrUserEmailTaken
	}
	role, ok := f.roles[user.RoleID]
	if !ok {
		return repository.ErrRoleNotFound
	}
	user.ID = len(f.users) + 1
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	stored := *user
	stored.Role = role
	f.users[user.ID] = &stored
	return nil
}

func (f *fakeUserRepository) Update(user *models.User) error {
	if _, ok := f.users[user.ID]; !ok {
		return repository.ErrUserNotFound
	}
	if f.emailTaken(user.Email, user.ID) {
		return repository.ErrUserEmailTaken
	}
	role, ok := f.roles[user.RoleID]
	if !ok {
		return repository.ErrRoleNotFound
	}
	user.UpdatedAt = time.Now()

	stored := *user
	stored.Role = role
	f.users[user.ID] = &stored
	return nil
}

func (f *fakeUserRepository) Delete(id int) error {
	if _, ok := f.users[id]; !ok {
		return repository.ErrUserNotFound
	}
	delete(f.users, id)
	return nil
}

func (f *fakeUserRepository) UpdateLastLogin(id int) error {
	f.lastLoginIDs = append(f.lastLoginIDs, id)
	return nil
}

func testLogger() *logger.Logger {
	return logger.New(&config.LoggerConfig{Level: "error", Format: "text"})
}

var (
	testAdminRole   = &models.Role{ID: 1, Name: "Администратор", Code: models.RoleCodeAdmin}
	testManagerRole = &models.Role{ID: 2, Name: "Менеджер", Code: "manager"}
)

func ptr[T any](v T) *T { return &v }

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

	if _, err := svc.GetByEmail("missing@nexera.test"); !errors.Is(err, repository.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
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

func TestUserService_List_PassesFilterToRepository(t *testing.T) {
	repo := newFakeUserRepository()
	svc := services.NewUserService(repo, testLogger())

	filter := services.ListUsersFilter{
		RoleID:   ptr(2),
		IsActive: ptr(false),
		Sort:     repository.UserSortEmail,
		Order:    repository.SortDesc,
		Limit:    10,
		Offset:   20,
	}
	if _, err := svc.List(filter); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := repo.lastFilter
	if got.RoleID == nil || *got.RoleID != 2 || got.IsActive == nil || *got.IsActive != false ||
		got.Sort != repository.UserSortEmail || got.Order != repository.SortDesc ||
		got.Limit != 10 || got.Offset != 20 {
		t.Errorf("filter not passed through correctly: %+v", got)
	}
}

func TestUserService_Create(t *testing.T) {
	repo := newFakeUserRepository().withRoles(testManagerRole)
	svc := services.NewUserService(repo, testLogger())

	user, err := svc.Create(models.CreateUserRequest{
		RoleID:   testManagerRole.ID,
		Name:     "Иван",
		Email:    "ivan@nexera.test",
		Password: "secret-password",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !user.IsActive {
		t.Error("new user must be active")
	}
	if user.Role == nil || user.Role.Code != testManagerRole.Code {
		t.Errorf("role not loaded: %+v", user.Role)
	}
	if user.PasswordHash == "secret-password" {
		t.Fatal("password stored in plain text")
	}
	if err := auth.CheckPassword(user.PasswordHash, "secret-password"); err != nil {
		t.Errorf("stored hash does not match password: %v", err)
	}
}

func TestUserService_Create_Invalid(t *testing.T) {
	valid := models.CreateUserRequest{RoleID: 2, Name: "Иван", Email: "ivan@nexera.test", Password: "secret-password"}

	tests := []struct {
		name   string
		mutate func(r *models.CreateUserRequest)
	}{
		{"empty name", func(r *models.CreateUserRequest) { r.Name = "" }},
		{"empty email", func(r *models.CreateUserRequest) { r.Email = "" }},
		{"malformed email", func(r *models.CreateUserRequest) { r.Email = "not-an-email" }},
		{"email with display name", func(r *models.CreateUserRequest) { r.Email = "Иван <ivan@nexera.test>" }},
		{"missing role", func(r *models.CreateUserRequest) { r.RoleID = 0 }},
		{"short password", func(r *models.CreateUserRequest) { r.Password = "short" }},
		{"password over bcrypt limit", func(r *models.CreateUserRequest) { r.Password = strings.Repeat("a", 73) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeUserRepository().withRoles(testManagerRole)
			svc := services.NewUserService(repo, testLogger())

			req := valid
			tt.mutate(&req)

			if _, err := svc.Create(req); !errors.Is(err, services.ErrInvalidUser) {
				t.Fatalf("expected ErrInvalidUser, got %v", err)
			}
			if len(repo.users) != 0 {
				t.Error("invalid user must not be saved")
			}
		})
	}
}

func TestUserService_Create_RepositoryErrors(t *testing.T) {
	existing := &models.User{ID: 1, RoleID: 2, Email: "taken@nexera.test", Role: testManagerRole}

	tests := []struct {
		name    string
		req     models.CreateUserRequest
		wantErr error
	}{
		{
			name:    "email taken",
			req:     models.CreateUserRequest{RoleID: 2, Name: "Иван", Email: "taken@nexera.test", Password: "secret-password"},
			wantErr: repository.ErrUserEmailTaken,
		},
		{
			name:    "role not found",
			req:     models.CreateUserRequest{RoleID: 99, Name: "Иван", Email: "ivan@nexera.test", Password: "secret-password"},
			wantErr: repository.ErrRoleNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := services.NewUserService(newFakeUserRepository(existing), testLogger())

			if _, err := svc.Create(tt.req); !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func newTestUser(id int, role *models.Role) *models.User {
	hash, _ := auth.HashPassword("old-password")
	return &models.User{
		ID:           id,
		RoleID:       role.ID,
		Role:         role,
		Name:         "Пользователь",
		Email:        "user" + string(rune('0'+id)) + "@nexera.test",
		PasswordHash: hash,
		IsActive:     true,
	}
}

func TestUserService_Update_PatchesOnlyProvidedFields(t *testing.T) {
	admin := newTestUser(1, testAdminRole)
	manager := newTestUser(2, testManagerRole)
	repo := newFakeUserRepository(admin, manager)
	svc := services.NewUserService(repo, testLogger())

	updated, err := svc.Update(admin.ID, manager.ID, models.UpdateUserRequest{Name: ptr("Новое имя")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.Name != "Новое имя" {
		t.Errorf("name not updated: %q", updated.Name)
	}
	if updated.Email != manager.Email || updated.RoleID != manager.RoleID || !updated.IsActive {
		t.Errorf("unchanged fields were modified: %+v", updated)
	}
	if updated.PasswordHash != manager.PasswordHash {
		t.Error("password hash must stay the same when password is not provided")
	}
}

func TestUserService_Update_Password(t *testing.T) {
	admin := newTestUser(1, testAdminRole)
	manager := newTestUser(2, testManagerRole)
	svc := services.NewUserService(newFakeUserRepository(admin, manager), testLogger())

	updated, err := svc.Update(admin.ID, manager.ID, models.UpdateUserRequest{Password: ptr("new-password")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := auth.CheckPassword(updated.PasswordHash, "new-password"); err != nil {
		t.Errorf("new password does not match stored hash: %v", err)
	}
}

func TestUserService_Update_ChangeRole(t *testing.T) {
	admin := newTestUser(1, testAdminRole)
	manager := newTestUser(2, testManagerRole)
	svc := services.NewUserService(newFakeUserRepository(admin, manager), testLogger())

	updated, err := svc.Update(admin.ID, manager.ID, models.UpdateUserRequest{RoleID: ptr(testAdminRole.ID)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updated.RoleID != testAdminRole.ID || updated.Role == nil || updated.Role.Code != testAdminRole.Code {
		t.Errorf("role not updated or not reloaded: role_id=%d role=%+v", updated.RoleID, updated.Role)
	}
}

func TestUserService_Update_CannotChangeOwnRole(t *testing.T) {
	admin := newTestUser(1, testAdminRole)
	repo := newFakeUserRepository(admin).withRoles(testManagerRole)
	svc := services.NewUserService(repo, testLogger())

	_, err := svc.Update(admin.ID, admin.ID, models.UpdateUserRequest{RoleID: ptr(testManagerRole.ID)})
	if !errors.Is(err, services.ErrCannotChangeOwnRole) {
		t.Fatalf("expected ErrCannotChangeOwnRole, got %v", err)
	}
	if repo.users[admin.ID].RoleID != testAdminRole.ID {
		t.Error("own role must not be changed")
	}
}

func TestUserService_Update_SelfWithoutRoleChange(t *testing.T) {
	admin := newTestUser(1, testAdminRole)
	svc := services.NewUserService(newFakeUserRepository(admin), testLogger())

	// role_id совпадает с текущим — это не смена роли, остальные поля себе менять можно.
	updated, err := svc.Update(admin.ID, admin.ID, models.UpdateUserRequest{
		RoleID: ptr(testAdminRole.ID),
		Name:   ptr("Главный админ"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "Главный админ" {
		t.Errorf("name not updated: %q", updated.Name)
	}
}

func TestUserService_Update_InvalidMergedUser(t *testing.T) {
	admin := newTestUser(1, testAdminRole)
	manager := newTestUser(2, testManagerRole)
	repo := newFakeUserRepository(admin, manager)
	svc := services.NewUserService(repo, testLogger())

	tests := []struct {
		name string
		req  models.UpdateUserRequest
	}{
		{"empty name", models.UpdateUserRequest{Name: ptr("")}},
		{"malformed email", models.UpdateUserRequest{Email: ptr("broken")}},
		{"short password", models.UpdateUserRequest{Password: ptr("short")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.Update(admin.ID, manager.ID, tt.req); !errors.Is(err, services.ErrInvalidUser) {
				t.Fatalf("expected ErrInvalidUser, got %v", err)
			}
			if *repo.users[manager.ID] != *manager {
				t.Error("invalid update must not be saved")
			}
		})
	}
}

func TestUserService_Update_NotFound(t *testing.T) {
	admin := newTestUser(1, testAdminRole)
	svc := services.NewUserService(newFakeUserRepository(admin), testLogger())

	_, err := svc.Update(admin.ID, 99, models.UpdateUserRequest{Name: ptr("Кто-то")})
	if !errors.Is(err, repository.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserService_Delete(t *testing.T) {
	admin := newTestUser(1, testAdminRole)
	manager := newTestUser(2, testManagerRole)
	repo := newFakeUserRepository(admin, manager)
	svc := services.NewUserService(repo, testLogger())

	if err := svc.Delete(admin.ID, manager.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := repo.users[manager.ID]; ok {
		t.Error("user was not deleted")
	}
}

func TestUserService_Delete_CannotDeleteSelf(t *testing.T) {
	admin := newTestUser(1, testAdminRole)
	repo := newFakeUserRepository(admin)
	svc := services.NewUserService(repo, testLogger())

	if err := svc.Delete(admin.ID, admin.ID); !errors.Is(err, services.ErrCannotDeleteSelf) {
		t.Fatalf("expected ErrCannotDeleteSelf, got %v", err)
	}
	if _, ok := repo.users[admin.ID]; !ok {
		t.Error("admin must not be deleted")
	}
}

func TestUserService_Delete_NotFound(t *testing.T) {
	admin := newTestUser(1, testAdminRole)
	svc := services.NewUserService(newFakeUserRepository(admin), testLogger())

	if err := svc.Delete(admin.ID, 99); !errors.Is(err, repository.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}
