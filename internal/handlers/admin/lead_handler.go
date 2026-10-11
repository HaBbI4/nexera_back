package admin

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"nexera_back/internal/httpresponse"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/repository"
	"nexera_back/internal/services"
)

// allowedLeadSortFields — белый список значений query-параметра ?sort=.
// Отдельный от repository.leadSortColumns: тот определяет SQL-колонки,
// этот — что разрешено принять от клиента (невалидное значение -> 400, а не тихий дефолт).
var allowedLeadSortFields = map[string]repository.LeadSortField{
	"created_at": repository.LeadSortCreatedAt,
	"name":       repository.LeadSortName,
}

// LeadHandler дает админ-панели просматривать и удалять заявки клиентов. Создания здесь
// нет: заявки оставляют клиенты через публичный POST /api/v1/leads
// (internal/handlers/public.LeadHandler). Кеша нет: заявки читаются только из админки.
type LeadHandler struct {
	leadService *services.LeadService
	log         *logger.Logger
}

// NewLeadHandler создает новый обработчик заявок админ-панели.
func NewLeadHandler(leadService *services.LeadService, log *logger.Logger) *LeadHandler {
	return &LeadHandler{
		leadService: leadService,
		log:         log,
	}
}

// GetLeads возвращает список заявок с фильтрацией, сортировкой и пагинацией.
//
// Как и в GET /api/v1/banners: limit/offset при некорректном значении молча заменяются
// значением по умолчанию, а фильтры и сортировка — строже, некорректное значение — это 400.
// По умолчанию сначала новые заявки (created_at desc) — в отличие от баннеров/FAQ, где
// порядок задает position, для входящих заявок естественен порядок "почтового ящика".
//
// @Summary      Список заявок
// @Description  Возвращает заявки клиентов (вместе с услугой, если она указана) с фильтрацией по услуге, источнику и периоду создания. Требует авторизации.
// @Tags         admin-leads
// @Produce      json
// @Param        service_id    query     int     false  "ID услуги — вернуть только заявки на эту услугу"
// @Param        source        query     string  false  "Источник заявки (например contact_form)"
// @Param        created_from  query     string  false  "Создана не раньше, RFC3339"
// @Param        created_to    query     string  false  "Создана не позже, RFC3339"
// @Param        sort          query     string  false  "Поле сортировки: created_at (по умолчанию) или name"
// @Param        order         query     string  false  "Направление сортировки: desc (по умолчанию) или asc"
// @Param        limit         query     int     false  "Количество записей (по умолчанию 50, максимум 100)"
// @Param        offset        query     int     false  "Смещение (по умолчанию 0)"
// @Success      200  {array}   models.Lead
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/leads [get]
func (h *LeadHandler) GetLeads(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	query := r.URL.Query()

	filter := services.ListLeadsFilter{
		Sort:  repository.LeadSortCreatedAt,
		Order: repository.SortDesc,
	}

	if serviceIDStr := query.Get("service_id"); serviceIDStr != "" {
		serviceID, err := strconv.Atoi(serviceIDStr)
		if err != nil || serviceID <= 0 {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid service_id")
			return
		}
		filter.ServiceID = &serviceID
	}

	if source := query.Get("source"); source != "" {
		filter.Source = &source
	}

	if createdFromStr := query.Get("created_from"); createdFromStr != "" {
		createdFrom, err := time.Parse(time.RFC3339, createdFromStr)
		if err != nil {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid created_from, expected RFC3339")
			return
		}
		filter.CreatedFrom = &createdFrom
	}

	if createdToStr := query.Get("created_to"); createdToStr != "" {
		createdTo, err := time.Parse(time.RFC3339, createdToStr)
		if err != nil {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid created_to, expected RFC3339")
			return
		}
		filter.CreatedTo = &createdTo
	}

	if filter.CreatedFrom != nil && filter.CreatedTo != nil && filter.CreatedTo.Before(*filter.CreatedFrom) {
		httpresponse.WriteError(w, http.StatusBadRequest, "created_to must not be before created_from")
		return
	}

	if sortStr := query.Get("sort"); sortStr != "" {
		sortField, ok := allowedLeadSortFields[sortStr]
		if !ok {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid sort, allowed: created_at, name")
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

	leads, err := h.leadService.List(filter)
	if err != nil {
		h.log.WithError(err).Error("failed to get leads")
		httpresponse.WriteError(w, http.StatusInternalServerError, "failed to get leads")
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, leads)
}

// GetLead возвращает заявку по ID.
// @Summary      Получить заявку
// @Description  Возвращает заявку клиента по ID вместе с услугой (service), если она указана и еще существует. Требует авторизации.
// @Tags         admin-leads
// @Produce      json
// @Param        id   path      int  true  "ID заявки"
// @Success      200  {object}  models.Lead
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/leads/{id} [get]
func (h *LeadHandler) GetLead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	leadID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || leadID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid lead id")
		return
	}

	var lead *models.Lead
	lead, err = h.leadService.GetByID(leadID)
	if err != nil {
		h.writeLeadError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, lead)
}

// DeleteLead удаляет заявку по ID.
// @Summary      Удалить заявку
// @Description  Удаляет заявку клиента по ID. Требует авторизации.
// @Tags         admin-leads
// @Param        id  path  int  true  "ID заявки"
// @Success      204  "No Content"
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/leads/{id} [delete]
func (h *LeadHandler) DeleteLead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	leadID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || leadID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid lead id")
		return
	}

	if err := h.leadService.Delete(leadID); err != nil {
		h.writeLeadError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// writeLeadError сопоставляет ошибки сервисного слоя с HTTP статус-кодами через errors.Is
// по сигнальным ошибкам — тот же подход, что writeBannerError в banner_handler.go.
func (h *LeadHandler) writeLeadError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrLeadNotFound):
		httpresponse.WriteError(w, http.StatusNotFound, "lead not found")
	default:
		h.log.WithError(err).Error("lead operation failed")
		httpresponse.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
