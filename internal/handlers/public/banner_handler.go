package public

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"nexera_back/internal/httpresponse"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/redis"
	"nexera_back/internal/repository"
	"nexera_back/internal/services"
)

// defaultCacheTTL время жизни кеша для одиночных GET-ответов этого пакета
const defaultCacheTTL = 15 * time.Minute

// allowedBannerSortFields — белый список значений query-параметра ?sort=.
// Отдельный от repository.bannerSortColumns: тот определяет SQL-колонки,
// этот — что разрешено принять от клиента (невалидное значение -> 400, а не тихий дефолт).
var allowedBannerSortFields = map[string]repository.BannerSortField{
	"position":   repository.BannerSortPosition,
	"created_at": repository.BannerSortCreatedAt,
}

// BannerHandler отдает баннеры сайта. Без авторизации — доступен любому клиенту.
type BannerHandler struct {
	bannerService *services.BannerService
	redisClient   *redis.Client
	log           *logger.Logger
}

// NewBannerHandler создает новый обработчик баннеров
func NewBannerHandler(bannerService *services.BannerService, redisClient *redis.Client, log *logger.Logger) *BannerHandler {
	return &BannerHandler{
		bannerService: bannerService,
		redisClient:   redisClient,
		log:           log,
	}
}

// GetBanners возвращает список баннеров сайта с фильтрацией и сортировкой.
//
// Параметры пагинации (limit/offset) ведут себя как во всех остальных списочных
// эндпоинтах проекта: некорректное значение молча заменяется значением по умолчанию.
// Параметры фильтрации/сортировки (page_id/sort/order) — новый для проекта вид параметров,
// и ведут себя строже: некорректное значение — это 400, а не тихая подмена дефолтом,
// чтобы клиент не думал, что получил отсортированный по X список, а на деле получил дефолт.
//
// @Summary      Список баннеров
// @Description  Возвращает баннеры, видимые прямо сейчас (активные и в окне показа), с фильтрацией по странице и сортировкой
// @Tags         banners
// @Produce      json
// @Param        page_id  query     int     false  "ID страницы — вернуть баннеры только этой страницы"
// @Param        sort     query     string  false  "Поле сортировки: position (по умолчанию) или created_at"
// @Param        order    query     string  false  "Направление сортировки: asc (по умолчанию) или desc"
// @Param        limit    query     int     false  "Количество записей (по умолчанию 50, максимум 100)"
// @Param        offset   query     int     false  "Смещение (по умолчанию 0)"
// @Success      200  {array}   models.Banner
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/banners [get]
func (h *BannerHandler) GetBanners(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	query := r.URL.Query()

	filter := services.ListBannersFilter{
		Sort:  repository.BannerSortPosition,
		Order: repository.SortAsc,
	}

	if pageIDStr := query.Get("page_id"); pageIDStr != "" {
		pageID, err := strconv.Atoi(pageIDStr)
		if err != nil || pageID <= 0 {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid page_id")
			return
		}
		filter.PageID = &pageID
	}

	if sortStr := query.Get("sort"); sortStr != "" {
		sortField, ok := allowedBannerSortFields[sortStr]
		if !ok {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid sort, allowed: position, created_at")
			return
		}
		filter.Sort = sortField
	}

	if orderStr := query.Get("order"); orderStr != "" {
		switch repository.SortOrder(orderStr) {
		case repository.SortAsc, repository.SortDesc:
			filter.Order = repository.SortOrder(orderStr)
		default:
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid order, allowed: asc, desc")
			return
		}
	}

	// Пагинация — по тому же правилу, что в остальных списочных эндпоинтах проекта:
	// некорректное значение не ошибка, а молчаливый возврат к дефолту.
	filter.Limit = 50 // По умолчанию
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			filter.Limit = l
		}
	}

	if offsetStr := query.Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	banners, err := h.bannerService.ListPublic(filter)
	if err != nil {
		h.log.WithError(err).Error("failed to get banners")
		httpresponse.WriteError(w, http.StatusInternalServerError, "failed to get banners")
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, banners)
}

// GetBanner возвращает один баннер по ID, если он видим прямо сейчас.
// @Summary      Получить баннер
// @Description  Возвращает баннер по ID, если он активен и находится в окне показа
// @Tags         banners
// @Produce      json
// @Param        id   path      int  true  "ID баннера"
// @Success      200  {object}  models.Banner
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/banners/{id} [get]
func (h *BannerHandler) GetBanner(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	bannerID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || bannerID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid banner id")
		return
	}

	cacheKey := redis.GenerateKey(redis.KeyPrefixBanner, strconv.Itoa(bannerID))
	var banner models.Banner
	if err := h.redisClient.Get(r.Context(), cacheKey, &banner); err == nil {
		h.log.WithField("banner_id", bannerID).Debug("Banner retrieved from cache")
		httpresponse.WriteJSON(w, http.StatusOK, &banner)
		return
	}

	bannerPtr, err := h.bannerService.GetPublicByID(bannerID)
	if err != nil {
		if errors.Is(err, repository.ErrBannerNotFound) {
			httpresponse.WriteError(w, http.StatusNotFound, "banner not found")
		} else {
			h.log.WithError(err).Error("failed to get banner")
			httpresponse.WriteError(w, http.StatusInternalServerError, "failed to get banner")
		}
		return
	}

	if err := h.redisClient.Set(r.Context(), cacheKey, bannerPtr, defaultCacheTTL); err != nil {
		h.log.WithError(err).Error("failed to cache banner")
	}

	httpresponse.WriteJSON(w, http.StatusOK, bannerPtr)
}
