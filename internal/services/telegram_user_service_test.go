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

// fakeTelegramUserRepository — реализация repository.TelegramUserRepository в памяти.
// Повторяет уникальность chat_id (telegram_users_chat_id_uq). GetByID возвращает копию —
// изменения сервиса не "сохраняются" без вызова Update.
type fakeTelegramUserRepository struct {
	users      map[int]*models.TelegramUser
	nextID     int
	lastFilter repository.TelegramUserFilter
}

var _ repository.TelegramUserRepository = (*fakeTelegramUserRepository)(nil)

func newFakeTelegramUserRepository(users ...*models.TelegramUser) *fakeTelegramUserRepository {
	repo := &fakeTelegramUserRepository{users: make(map[int]*models.TelegramUser)}
	for _, u := range users {
		copied := *u
		repo.users[u.ID] = &copied
		if u.ID > repo.nextID {
			repo.nextID = u.ID
		}
	}
	return repo
}

func (f *fakeTelegramUserRepository) chatIDTaken(chatID int64, exceptID int) bool {
	for _, u := range f.users {
		if u.ChatID == chatID && u.ID != exceptID {
			return true
		}
	}
	return false
}

func (f *fakeTelegramUserRepository) List(filter repository.TelegramUserFilter) ([]*models.TelegramUser, error) {
	f.lastFilter = filter

	var result []*models.TelegramUser
	for _, u := range f.users {
		if filter.IsActive != nil && u.IsActive != *filter.IsActive {
			continue
		}
		result = append(result, u)
	}
	return result, nil
}

func (f *fakeTelegramUserRepository) GetByID(id int) (*models.TelegramUser, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, repository.ErrTelegramUserNotFound
	}
	copied := *u
	return &copied, nil
}

func (f *fakeTelegramUserRepository) GetByChatID(chatID int64) (*models.TelegramUser, error) {
	for _, u := range f.users {
		if u.ChatID == chatID {
			copied := *u
			return &copied, nil
		}
	}
	return nil, repository.ErrTelegramUserNotFound
}

func (f *fakeTelegramUserRepository) Create(user *models.TelegramUser) error {
	if f.chatIDTaken(user.ChatID, 0) {
		return repository.ErrTelegramUserChatIDTaken
	}
	f.nextID++
	user.ID = f.nextID
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	stored := *user
	f.users[user.ID] = &stored
	return nil
}

func (f *fakeTelegramUserRepository) Update(user *models.TelegramUser) error {
	if _, ok := f.users[user.ID]; !ok {
		return repository.ErrTelegramUserNotFound
	}
	if f.chatIDTaken(user.ChatID, user.ID) {
		return repository.ErrTelegramUserChatIDTaken
	}
	user.UpdatedAt = time.Now()
	stored := *user
	f.users[user.ID] = &stored
	return nil
}

func (f *fakeTelegramUserRepository) Delete(id int) error {
	if _, ok := f.users[id]; !ok {
		return repository.ErrTelegramUserNotFound
	}
	delete(f.users, id)
	return nil
}

func newTelegramUserFixtures() *fakeTelegramUserRepository {
	return newFakeTelegramUserRepository(
		&models.TelegramUser{ID: 1, ChatID: 111, Name: "Владелец", IsActive: true},
		&models.TelegramUser{ID: 2, ChatID: 222, Name: "Бывший сотрудник", IsActive: false},
	)
}

func TestTelegramUserService_List_IncludesInactive(t *testing.T) {
	repo := newTelegramUserFixtures()
	svc := services.NewTelegramUserService(repo, testLogger())

	users, err := svc.List(services.ListTelegramUsersFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if repo.lastFilter.IsActive != nil {
		t.Error("admin list must not force is_active")
	}
}

func TestTelegramUserService_GetActiveUsers_ForcesActive(t *testing.T) {
	repo := newTelegramUserFixtures()
	svc := services.NewTelegramUserService(repo, testLogger())

	users, err := svc.GetActiveUsers()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 1 || users[0].ID != 1 {
		t.Fatalf("expected only active user 1, got %+v", users)
	}
	if repo.lastFilter.IsActive == nil || !*repo.lastFilter.IsActive {
		t.Error("GetActiveUsers must filter is_active = true")
	}
}

func TestTelegramUserService_GetByID_NotFound(t *testing.T) {
	svc := services.NewTelegramUserService(newTelegramUserFixtures(), testLogger())

	if _, err := svc.GetByID(99); !errors.Is(err, repository.ErrTelegramUserNotFound) {
		t.Fatalf("expected ErrTelegramUserNotFound, got %v", err)
	}
}

func TestTelegramUserService_Create_DefaultsToActive(t *testing.T) {
	svc := services.NewTelegramUserService(newFakeTelegramUserRepository(), testLogger())

	user, err := svc.Create(models.CreateTelegramUserRequest{ChatID: 333, Name: "Менеджер"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !user.IsActive {
		t.Error("user without is_active must be created active")
	}
	if user.ID == 0 {
		t.Error("expected ID to be assigned")
	}
}

func TestTelegramUserService_Create_ExplicitInactive(t *testing.T) {
	svc := services.NewTelegramUserService(newFakeTelegramUserRepository(), testLogger())

	user, err := svc.Create(models.CreateTelegramUserRequest{ChatID: 333, Name: "Менеджер", IsActive: ptr(false)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.IsActive {
		t.Error("explicit is_active=false must be kept")
	}
}

func TestTelegramUserService_Create_GroupChatID(t *testing.T) {
	svc := services.NewTelegramUserService(newFakeTelegramUserRepository(), testLogger())

	if _, err := svc.Create(models.CreateTelegramUserRequest{ChatID: -1001234567890, Name: "Чат отдела продаж"}); err != nil {
		t.Fatalf("negative chat_id (group) must be allowed, got %v", err)
	}
}

func TestTelegramUserService_Create_Invalid(t *testing.T) {
	tests := []struct {
		name string
		req  models.CreateTelegramUserRequest
	}{
		{"missing chat_id", models.CreateTelegramUserRequest{Name: "Менеджер"}},
		{"missing name", models.CreateTelegramUserRequest{ChatID: 333}},
		{"name too long", models.CreateTelegramUserRequest{ChatID: 333, Name: strings.Repeat("я", 256)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := services.NewTelegramUserService(newFakeTelegramUserRepository(), testLogger())
			if _, err := svc.Create(tt.req); !errors.Is(err, services.ErrInvalidTelegramUser) {
				t.Fatalf("expected ErrInvalidTelegramUser, got %v", err)
			}
		})
	}
}

func TestTelegramUserService_Create_ChatIDTaken(t *testing.T) {
	svc := services.NewTelegramUserService(newTelegramUserFixtures(), testLogger())

	_, err := svc.Create(models.CreateTelegramUserRequest{ChatID: 111, Name: "Дубль"})
	if !errors.Is(err, repository.ErrTelegramUserChatIDTaken) {
		t.Fatalf("expected ErrTelegramUserChatIDTaken, got %v", err)
	}
}

func TestTelegramUserService_Update_PartialKeepsOtherFields(t *testing.T) {
	repo := newTelegramUserFixtures()
	svc := services.NewTelegramUserService(repo, testLogger())

	user, err := svc.Update(1, models.UpdateTelegramUserRequest{IsActive: ptr(false)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.IsActive {
		t.Error("expected user to be deactivated")
	}
	if user.ChatID != 111 || user.Name != "Владелец" {
		t.Errorf("fields not in request must be kept, got %+v", user)
	}
	if stored, _ := repo.GetByID(1); stored.IsActive {
		t.Error("update must be persisted")
	}
}

func TestTelegramUserService_Update_InvalidNotPersisted(t *testing.T) {
	repo := newTelegramUserFixtures()
	svc := services.NewTelegramUserService(repo, testLogger())

	if _, err := svc.Update(1, models.UpdateTelegramUserRequest{Name: ptr("")}); !errors.Is(err, services.ErrInvalidTelegramUser) {
		t.Fatalf("expected ErrInvalidTelegramUser, got %v", err)
	}
	if stored, _ := repo.GetByID(1); stored.Name != "Владелец" {
		t.Error("invalid update must not be persisted")
	}
}

func TestTelegramUserService_Update_ChatIDTaken(t *testing.T) {
	svc := services.NewTelegramUserService(newTelegramUserFixtures(), testLogger())

	_, err := svc.Update(1, models.UpdateTelegramUserRequest{ChatID: ptr(int64(222))})
	if !errors.Is(err, repository.ErrTelegramUserChatIDTaken) {
		t.Fatalf("expected ErrTelegramUserChatIDTaken, got %v", err)
	}
}

func TestTelegramUserService_Update_SameChatIDAllowed(t *testing.T) {
	svc := services.NewTelegramUserService(newTelegramUserFixtures(), testLogger())

	if _, err := svc.Update(1, models.UpdateTelegramUserRequest{ChatID: ptr(int64(111)), Name: ptr("Новое имя")}); err != nil {
		t.Fatalf("keeping own chat_id must be allowed, got %v", err)
	}
}

func TestTelegramUserService_Update_NotFound(t *testing.T) {
	svc := services.NewTelegramUserService(newTelegramUserFixtures(), testLogger())

	if _, err := svc.Update(99, models.UpdateTelegramUserRequest{Name: ptr("x")}); !errors.Is(err, repository.ErrTelegramUserNotFound) {
		t.Fatalf("expected ErrTelegramUserNotFound, got %v", err)
	}
}

func TestTelegramUserService_Delete(t *testing.T) {
	svc := services.NewTelegramUserService(newTelegramUserFixtures(), testLogger())

	if err := svc.Delete(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := svc.GetByChatID(111); !errors.Is(err, repository.ErrTelegramUserNotFound) {
		t.Fatalf("deleted user must lose bot access, got %v", err)
	}
	if err := svc.Delete(1); !errors.Is(err, repository.ErrTelegramUserNotFound) {
		t.Fatalf("expected ErrTelegramUserNotFound on second delete, got %v", err)
	}
}
