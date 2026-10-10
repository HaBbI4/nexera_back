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

// allowedServiceSortFields — белый список значений query-параметра ?sort= для админского
// списка услуг (тот же набор, что в публичном internal/handlers/public).
var allowedServiceSortFields = map[string]repository.ServiceSortField{
	"position":   repository.ServiceSortPosition,
	"price":      repository.ServiceSortPrice,
	"name":       repository.ServiceSortName,
	"created_at": repository.ServiceSortCreatedAt,
}

// ServiceHandler управляет услугами компании из админ-панели: просмотр, создание,
// обновление, удаление. В отличие от публичного internal/handlers/public.ServiceHandler,
// чтение здесь показывает все услуги, включая неопубликованные, и без кеша. Изображений
// у услуг нет, поэтому тело запроса — обычный application/json, как у страниц.
type ServiceHandler struct {
	serviceService *services.ServiceService
	redisClient    *redis.Client
	log            *logger.Logger
}

// NewServiceHandler создает новый обработчик услуг админ-панели.
func NewServiceHandler(serviceService *services.ServiceService, redisClient *redis.Client, log *logger.Logger) *ServiceHandler {
	return &ServiceHandler{
		serviceService: serviceService,
		redisClient:    redisClient,
		log:            log,
	}
}

// GetServices возвращает список всех услуг для админ-панели с фильтрацией, сортировкой
// и пагинацией — вне зависимости от публикации.
//
// Как и в публичном GET /api/v1/services: limit/offset при некорректном значении молча
// заменяются значением по умолчанию, а фильтры и сортировка — строже, некорректное значение — 400.
//
// @Summary      Список услуг (админка)
// @Description  Возвращает все услуги, включая неопубликованные. Требует авторизации.
// @Tags         admin-services
// @Produce      json
// @Param        is_published  query     bool    false  "true — только опубликованные, false — только скрытые"
// @Param        price_from    query     int     false  "Цена не меньше (включительно)"
// @Param        price_to      query     int     false  "Цена не больше (включительно)"
// @Param        sort          query     string  false  "Поле сортировки: position (по умолчанию), price, name или created_at"
// @Param        order         query     string  false  "Направление сортировки: asc (по умолчанию) или desc"
// @Param        limit         query     int     false  "Количество записей (по умолчанию 50, максимум 100)"
// @Param        offset        query     int     false  "Смещение (по умолчанию 0)"
// @Success      200  {array}   models.Service
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/services [get]
func (h *ServiceHandler) GetServices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	query := r.URL.Query()

	filter := services.ListAdminServicesFilter{
		Sort:  repository.ServiceSortPosition,
		Order: repository.SortAsc,
	}

	if isPublishedStr := query.Get("is_published"); isPublishedStr != "" {
		isPublished, err := strconv.ParseBool(isPublishedStr)
		if err != nil {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid is_published")
			return
		}
		filter.IsPublished = &isPublished
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

	servicesList, err := h.serviceService.List(filter)
	if err != nil {
		h.log.WithError(err).Error("failed to get services")
		httpresponse.WriteError(w, http.StatusInternalServerError, "failed to get services")
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, servicesList)
}

// GetService возвращает услугу по ID для админ-панели — вне зависимости от публикации.
// @Summary      Получить услугу (админка)
// @Description  Возвращает услугу по ID, включая неопубликованную. Требует авторизации.
// @Tags         admin-services
// @Produce      json
// @Param        id   path      int  true  "ID услуги"
// @Success      200  {object}  models.Service
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/services/{id} [get]
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

	service, err := h.serviceService.GetByID(serviceID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, service)
}

// CreateService создает услугу.
// @Summary      Создать услугу
// @Description  Создает новую услугу. Без price — "цена по запросу". Требует авторизации.
// @Tags         admin-services
// @Accept       json
// @Produce      json
// @Param        service  body      models.CreateServiceRequest  true  "Данные услуги"
// @Success      201  {object}  models.Service
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/services [post]
func (h *ServiceHandler) CreateService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	var req models.CreateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	service, err := h.serviceService.Create(req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusCreated, service)
}

// UpdateService частично обновляет услугу по ID.
// @Summary      Обновить услугу
// @Description  Частично обновляет услугу по ID. Требует авторизации.
// @Tags         admin-services
// @Accept       json
// @Produce      json
// @Param        id       path      int                          true  "ID услуги"
// @Param        service  body      models.UpdateServiceRequest  true  "Поля для обновления"
// @Success      200  {object}  models.Service
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/services/{id} [patch]
func (h *ServiceHandler) UpdateService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	serviceID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || serviceID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}

	var req models.UpdateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req == (models.UpdateServiceRequest{}) {
		httpresponse.WriteError(w, http.StatusBadRequest, "at least one field is required")
		return
	}

	service, err := h.serviceService.Update(serviceID, req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	// Инвалидация кеша: следующий публичный GET /services/{id} перечитает из БД.
	h.invalidateServiceCache(r, serviceID)

	httpresponse.WriteJSON(w, http.StatusOK, service)
}

// DeleteService удаляет услугу по ID. Заявки на эту услугу не удаляются — у них
// обнуляется service_id (leads.service_id -> services(id) ON DELETE SET NULL).
// @Summary      Удалить услугу
// @Description  Удаляет услугу по ID. Заявки на нее остаются, но без ссылки на услугу. Требует авторизации.
// @Tags         admin-services
// @Param        id  path  int  true  "ID услуги"
// @Success      204  "No Content"
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/services/{id} [delete]
func (h *ServiceHandler) DeleteService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	serviceID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || serviceID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid service id")
		return
	}

	if err := h.serviceService.Delete(serviceID); err != nil {
		h.writeServiceError(w, err)
		return
	}

	h.invalidateServiceCache(r, serviceID)

	w.WriteHeader(http.StatusNoContent)
}

// invalidateServiceCache сбрасывает кеш публичного GET /api/v1/services/{id}.
func (h *ServiceHandler) invalidateServiceCache(r *http.Request, serviceID int) {
	cacheKey := redis.GenerateKey(redis.KeyPrefixService, strconv.Itoa(serviceID))
	if err := h.redisClient.Delete(r.Context(), cacheKey); err != nil {
		h.log.WithError(err).Error("failed to invalidate service cache")
	}
}

// writeServiceError сопоставляет ошибки сервисного слоя с HTTP статус-кодами через errors.Is
// по сигнальным ошибкам — тот же подход, что writeBannerError в banner_handler.go.
func (h *ServiceHandler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidService):
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, repository.ErrServiceNotFound):
		httpresponse.WriteError(w, http.StatusNotFound, "service not found")
	default:
		h.log.WithError(err).Error("service operation failed")
		httpresponse.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
