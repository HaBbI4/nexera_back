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

// PageHandler управляет страницами из админ-панели: создание, обновление, удаление.
// Чтение (список/по ID с баннерами) публичное и без авторизации — см.
// internal/handlers/public.PageHandler. Страницы не несут изображений, поэтому, в отличие
// от баннеров, тело запроса здесь обычный application/json, без multipart.
type PageHandler struct {
	pageService *services.PageService
	redisClient *redis.Client
	log         *logger.Logger
}

// NewPageHandler создает новый обработчик страниц админ-панели.
func NewPageHandler(pageService *services.PageService, redisClient *redis.Client, log *logger.Logger) *PageHandler {
	return &PageHandler{
		pageService: pageService,
		redisClient: redisClient,
		log:         log,
	}
}

// CreatePage создает страницу.
// @Summary      Создать страницу
// @Description  Создает новую страницу. Требует авторизации.
// @Tags         admin-pages
// @Accept       json
// @Produce      json
// @Param        page  body      models.CreatePageRequest  true  "Данные страницы"
// @Success      201  {object}  models.Page
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/pages [post]
func (h *PageHandler) CreatePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	var req models.CreatePageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	page, err := h.pageService.Create(req)
	if err != nil {
		h.writePageError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusCreated, page)
}

// UpdatePage частично обновляет страницу по ID.
// @Summary      Обновить страницу
// @Description  Частично обновляет страницу по ID. Требует авторизации.
// @Tags         admin-pages
// @Accept       json
// @Produce      json
// @Param        id    path      int                       true  "ID страницы"
// @Param        page  body      models.UpdatePageRequest  true  "Поля для обновления"
// @Success      200  {object}  models.Page
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/pages/{id} [patch]
func (h *PageHandler) UpdatePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	pageID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || pageID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid page id")
		return
	}

	var req models.UpdatePageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req == (models.UpdatePageRequest{}) {
		httpresponse.WriteError(w, http.StatusBadRequest, "at least one field is required")
		return
	}

	page, err := h.pageService.Update(pageID, req)
	if err != nil {
		h.writePageError(w, err)
		return
	}

	// Инвалидация кеша: следующий публичный GET /pages/{id} перечитает из БД.
	cacheKey := redis.GenerateKey(redis.KeyPrefixPage, strconv.Itoa(pageID))
	if err := h.redisClient.Delete(r.Context(), cacheKey); err != nil {
		h.log.WithError(err).Error("failed to invalidate page cache")
	}

	httpresponse.WriteJSON(w, http.StatusOK, page)
}

// DeletePage удаляет страницу по ID. Внимание: это каскадно удалит и все баннеры
// этой страницы (banners.page_id -> pages(id) ON DELETE CASCADE).
// @Summary      Удалить страницу
// @Description  Удаляет страницу по ID вместе со всеми ее баннерами (ON DELETE CASCADE). Требует авторизации.
// @Tags         admin-pages
// @Param        id  path  int  true  "ID страницы"
// @Success      204  "No Content"
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/pages/{id} [delete]
func (h *PageHandler) DeletePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	pageID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || pageID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid page id")
		return
	}

	if err := h.pageService.Delete(pageID); err != nil {
		h.writePageError(w, err)
		return
	}

	cacheKey := redis.GenerateKey(redis.KeyPrefixPage, strconv.Itoa(pageID))
	if err := h.redisClient.Delete(r.Context(), cacheKey); err != nil {
		h.log.WithError(err).Error("failed to invalidate page cache")
	}

	w.WriteHeader(http.StatusNoContent)
}

// writePageError сопоставляет ошибки сервисного слоя с HTTP статус-кодами через errors.Is
// по сигнальным ошибкам — тот же подход, что writeBannerError в banner_handler.go.
func (h *PageHandler) writePageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidPage):
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, repository.ErrPageNotFound):
		httpresponse.WriteError(w, http.StatusNotFound, "page not found")
	case errors.Is(err, repository.ErrPageSlugTaken):
		httpresponse.WriteError(w, http.StatusConflict, "page slug already taken")
	default:
		h.log.WithError(err).Error("page operation failed")
		httpresponse.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
