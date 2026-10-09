package admin

import (
	"encoding/json"
	"net/http"
	"time"

	"nexera_back/internal/auth"
	"nexera_back/internal/httpresponse"
	"nexera_back/internal/logger"
	"nexera_back/internal/middleware"
	"nexera_back/internal/models"
	"nexera_back/internal/services"
)

// AuthHandler обрабатывает вход пользователей админ-панели и выдачу JWT токенов.
type AuthHandler struct {
	userService *services.UserService
	jwtSecret   string
	jwtTTL      time.Duration
	log         *logger.Logger
}

// NewAuthHandler создает новый обработчик аутентификации админ-панели
func NewAuthHandler(userService *services.UserService, jwtSecret string, jwtTTL time.Duration, log *logger.Logger) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		jwtSecret:   jwtSecret,
		jwtTTL:      jwtTTL,
		log:         log,
	}
}

// LoginRequest представляет запрос на вход в админ-панель
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse представляет ответ на успешный вход
type LoginResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

// Login проверяет email/пароль и выдает JWT токен для доступа к админ-панели.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		httpresponse.WriteError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := h.userService.GetByEmail(req.Email)
	if err != nil {
		httpresponse.WriteError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if !user.IsActive {
		httpresponse.WriteError(w, http.StatusForbidden, "user is disabled")
		return
	}

	if err := auth.CheckPassword(user.PasswordHash, req.Password); err != nil {
		httpresponse.WriteError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := auth.GenerateToken(user, h.jwtSecret, h.jwtTTL)
	if err != nil {
		h.log.WithError(err).Error("failed to generate token")
		httpresponse.WriteError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	if err := h.userService.UpdateLastLogin(user.ID); err != nil {
		h.log.WithError(err).Warn("failed to update last_login_at")
	}

	user.PasswordHash = ""
	httpresponse.WriteJSON(w, http.StatusOK, LoginResponse{Token: token, User: *user})
}

// Me возвращает данные аутентифицированного пользователя. Требует RequireAuth.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	authUser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		httpresponse.WriteError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}

	user, err := h.userService.GetByID(authUser.ID)
	if err != nil {
		httpresponse.WriteError(w, http.StatusNotFound, "user not found")
		return
	}

	user.PasswordHash = ""
	httpresponse.WriteJSON(w, http.StatusOK, user)
}
