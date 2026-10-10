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

// allowedServiceSortFields — белый список значений query-параметра ?sort=.
// Отдельный от repository.serviceSortColumns: тот определяет SQL-колонки,
// этот — что разрешено принять от клиента (невалидное значение -> 400, а не тихий дефолт).
var allowedServiceSortFields = map[string]repository.ServiceSortField{
	"position":   repository.ServiceSortPosition,
	"price":      repository.ServiceSortPrice,
	"name":       repository.ServiceSortName,
	"created_at": repository.ServiceSortCreatedAt,
}

// ServiceHandler отдает услуги компании. Без авторизации — доступен любому клиенту.
type ServiceHandler struct {
	serviceService *services.ServiceService
	redisClient    *redis.Client
	log            *logger.Logger
}

// NewServiceHandler создает новый обработчик услуг.
func NewServiceHandler(serviceService *services.ServiceService, redisClient *redis.Client, log *logger.Logger) *ServiceHandler {
	return &ServiceHandler{
		serviceService: serviceService,
		redisClient:    redisClient,
		log:            log,
	}
}

// GetServices возвращает список опубликованных услуг с фильтрацией по цене, сортировкой
// и пагинацией.
//
// Как и в GET /api/v1/banners: limit/offset при некорректном значении молча заменяются
// значением по умолчанию, а фильтры и сортировка — строже, некорректное значение — это 400.
//
// @Summary      Список услуг
// @Description  Возвращает опубликованные услуги с фильтрацией по цене, сортировкой и пагинацией. Услуги без цены ("по запросу") под ценовой фильтр не попадают, а при сортировке по цене всегда идут в конце.
// @Tags         services
// @Produce      json
// @Param        price_from  query     int     false  "Цена не меньше (включительно)"
// @Param        price_to    query     int     false  "Цена не больше (включительно)"
// @Param        sort        query     string  false  "Поле сортировки: position (по умолчанию), price, name или created_at"
// @Param        order       query     string  false  "Направление сортировки: asc (по умолчанию) или desc"
// @Param        limit       query     int     false  "Количество записей (по умолчанию 50, максимум 100)"
// @Param        offset      query     int     false  "Смещение (по умолчанию 0)"
// @Success      200  {array}   models.Service
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/services [get]
func (h *ServiceHandler) GetServices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	query := r.URL.Query()

	filter := services.ListServicesFilter{
		Sort:  repository.ServiceSortPosition,
		Order: repository.SortAsc,
	}

	if priceFromStr := query.Get("price_from"); priceFromStr != "" {
		priceFrom, err := strconv.Atoi(priceFromStr)
		if err != nil || priceFrom < 0 {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid price_from")
			return
		}
		filter.PriceFrom = &priceFrom
	}

	if priceToStr := query.Get("price_to"); priceToStr != "" {
		priceTo, err := strconv.Atoi(priceToStr)
		if err != nil || priceTo < 0 {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid price_to")
			return
		}
		filter.PriceTo = &priceTo
	}

	if filter.PriceFrom != nil && filter.PriceTo != nil && *filter.PriceTo < *filter.PriceFrom {
		httpresponse.WriteError(w, http.StatusBadRequest, "price_to must not be less than price_from")
		return
	}

	if sortStr := query.Get("sort"); sortStr != "" {
		sortField, ok := allowedServiceSortFields[sortStr]
		if !ok {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid sort, allowed: position, price, name, created_at")
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

	servicesList, err := h.serviceService.ListPublic(filter)
	if err != nil {
		h.log.WithError(err).Error("failed to get services")
		httpresponse.WriteError(w, http.StatusInternalServerError, "failed to get services")
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, servicesList)
}

// GetService возвращает одну услугу по ID, если она опубликована.
// @Summary      Получить услугу
// @Description  Возвращает услугу по ID, если она опубликована
// @Tags         services
// @Produce      json
// @Param        id   path      int  true  "ID услуги"
// @Success      200  {object}  models.Service
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/services/{id} [get]
func (h *ServiceHandler) GetService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	serviceID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || serviceID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}

	cacheKey := redis.GenerateKey(redis.KeyPrefixService, strconv.Itoa(serviceID))
	var service models.Service
	if err := h.redisClient.Get(r.Context(), cacheKey, &service); err == nil {
		h.log.WithField("service_id", serviceID).Debug("Service retrieved from cache")
		httpresponse.WriteJSON(w, http.StatusOK, &service)
		return
	}

	servicePtr, err := h.serviceService.GetPublicByID(serviceID)
	if err != nil {
		if errors.Is(err, repository.ErrServiceNotFound) {
			httpresponse.WriteError(w, http.StatusNotFound, "service not found")
		} else {
			h.log.WithError(err).Error("failed to get service")
			httpresponse.WriteError(w, http.StatusInternalServerError, "failed to get service")
		}
		return
	}

	if err := h.redisClient.Set(r.Context(), cacheKey, servicePtr, defaultCacheTTL); err != nil {
		h.log.WithError(err).Error("failed to cache service")
	}

	httpresponse.WriteJSON(w, http.StatusOK, servicePtr)
}
