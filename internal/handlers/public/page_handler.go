package public

import (
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

// PageHandler отдает страницы сайта. Без авторизации — доступен любому клиенту.
type PageHandler struct {
	pageService *services.PageService
	redisClient *redis.Client
	log         *logger.Logger
}

// NewPageHandler создает новый обработчик страниц.
func NewPageHandler(pageService *services.PageService, redisClient *redis.Client, log *logger.Logger) *PageHandler {
	return &PageHandler{
		pageService: pageService,
		redisClient: redisClient,
		log:         log,
	}
}

// GetPages возвращает список всех страниц сайта.
// @Summary      Список страниц
// @Description  Возвращает все страницы сайта (без баннеров)
// @Tags         pages
// @Produce      json
// @Success      200  {array}   models.Page
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/pages [get]
func (h *PageHandler) GetPages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	pages, err := h.pageService.ListPublic()
	if err != nil {
		h.log.WithError(err).Error("failed to get pages")
		httpresponse.WriteError(w, http.StatusInternalServerError, "failed to get pages")
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, pages)
}

// GetPage возвращает одну страницу по ID вместе с ее баннерами, видимыми прямо сейчас,
// отсортированными по позиции.
// @Summary      Получить страницу
// @Description  Возвращает страницу по ID вместе с баннерами (активными и в окне показа), отсортированными по позиции
// @Tags         pages
// @Produce      json
// @Param        id   path      int  true  "ID страницы"
// @Success      200  {object}  models.Page
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/pages/{id} [get]
func (h *PageHandler) GetPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	pageID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || pageID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid page id")
		return
	}

	cacheKey := redis.GenerateKey(redis.KeyPrefixPage, strconv.Itoa(pageID))
	var page models.Page
	if err := h.redisClient.Get(r.Context(), cacheKey, &page); err == nil {
		h.log.WithField("page_id", pageID).Debug("Page retrieved from cache")
		httpresponse.WriteJSON(w, http.StatusOK, &page)
		return
	}

	pagePtr, err := h.pageService.GetPublicByIDWithBanners(pageID)
	if err != nil {
		if errors.Is(err, repository.ErrPageNotFound) {
			httpresponse.WriteError(w, http.StatusNotFound, "page not found")
		} else {
			h.log.WithError(err).Error("failed to get page")
			httpresponse.WriteError(w, http.StatusInternalServerError, "failed to get page")
		}
		return
	}

	if err := h.redisClient.Set(r.Context(), cacheKey, pagePtr, defaultCacheTTL); err != nil {
		h.log.WithError(err).Error("failed to cache page")
	}

	httpresponse.WriteJSON(w, http.StatusOK, pagePtr)
}
