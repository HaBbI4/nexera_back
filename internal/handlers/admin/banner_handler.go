package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"nexera_back/internal/httpresponse"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/redis"
	"nexera_back/internal/repository"
	"nexera_back/internal/services"
)

// BannerHandler управляет баннерами из админ-панели: создание, обновление, удаление.
// Чтение (список/по ID) публичное и без авторизации — см. internal/handlers/public.BannerHandler.
type BannerHandler struct {
	bannerService *services.BannerService
	redisClient   *redis.Client
	log           *logger.Logger
}

// NewBannerHandler создает новый обработчик баннеров админ-панели
func NewBannerHandler(bannerService *services.BannerService, redisClient *redis.Client, log *logger.Logger) *BannerHandler {
	return &BannerHandler{
		bannerService: bannerService,
		redisClient:   redisClient,
		log:           log,
	}
}

// CreateBanner создает баннер.
// @Summary      Создать баннер
// @Description  Создает новый баннер. Требует авторизации.
// @Tags         admin-banners
// @Accept       json
// @Produce      json
// @Param        request  body      models.CreateBannerRequest  true  "Данные баннера"
// @Success      201  {object}  models.Banner
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/banners [post]
func (h *BannerHandler) CreateBanner(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	var req models.CreateBannerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	banner, err := h.bannerService.Create(req)
	if err != nil {
		h.writeBannerError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusCreated, banner)
}

// UpdateBanner частично обновляет баннер — в теле передаются только изменяемые поля,
// остальные сохраняют текущее значение (см. BannerService.Update).
// @Summary      Обновить баннер
// @Description  Частично обновляет баннер по ID. Требует авторизации.
// @Tags         admin-banners
// @Accept       json
// @Produce      json
// @Param        id       path      int                         true  "ID баннера"
// @Param        request  body      models.UpdateBannerRequest  true  "Поля для обновления"
// @Success      200  {object}  models.Banner
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/banners/{id} [patch]
func (h *BannerHandler) UpdateBanner(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	bannerID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || bannerID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid banner id")
		return
	}

	var req models.UpdateBannerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req == (models.UpdateBannerRequest{}) {
		httpresponse.WriteError(w, http.StatusBadRequest, "at least one field is required")
		return
	}

	banner, err := h.bannerService.Update(bannerID, req)
	if err != nil {
		h.writeBannerError(w, err)
		return
	}

	// Инвалидация кеша: следующий публичный GET /banners/{id} перечитает из БД.
	cacheKey := redis.GenerateKey(redis.KeyPrefixBanner, strconv.Itoa(bannerID))
	if err := h.redisClient.Delete(r.Context(), cacheKey); err != nil {
		h.log.WithError(err).Error("failed to invalidate banner cache")
	}

	httpresponse.WriteJSON(w, http.StatusOK, banner)
}

// DeleteBanner удаляет баннер по ID.
// @Summary      Удалить баннер
// @Description  Удаляет баннер по ID. Требует авторизации.
// @Tags         admin-banners
// @Param        id  path  int  true  "ID баннера"
// @Success      204  "No Content"
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/banners/{id} [delete]
func (h *BannerHandler) DeleteBanner(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	bannerID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || bannerID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid banner id")
		return
	}

	if err := h.bannerService.Delete(bannerID); err != nil {
		h.writeBannerError(w, err)
		return
	}

	cacheKey := redis.GenerateKey(redis.KeyPrefixBanner, strconv.Itoa(bannerID))
	if err := h.redisClient.Delete(r.Context(), cacheKey); err != nil {
		h.log.WithError(err).Error("failed to invalidate banner cache")
	}

	w.WriteHeader(http.StatusNoContent)
}

// writeBannerError сопоставляет ошибки сервисного слоя с HTTP статус-кодами через errors.Is
// по сигнальным ошибкам (ErrInvalidBanner/ErrBannerNotFound/ErrPageNotFound), а не разбором
// текста сообщения — это надежнее и не ломается при изменении формулировок.
func (h *BannerHandler) writeBannerError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidBanner):
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, repository.ErrBannerNotFound):
		httpresponse.WriteError(w, http.StatusNotFound, "banner not found")
	case errors.Is(err, repository.ErrPageNotFound):
		httpresponse.WriteError(w, http.StatusBadRequest, "page not found")
	default:
		h.log.WithError(err).Error("banner operation failed")
		httpresponse.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
