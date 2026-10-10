package public

import (
	"encoding/json"
	"errors"
	"net/http"

	"nexera_back/internal/httpresponse"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/repository"
	"nexera_back/internal/services"
)

// maxLeadBodyBytes ограничивает размер тела публичного POST /leads: эндпоинт без
// авторизации, а самая длинная заявка (message до 5000 символов) укладывается в ~20KB.
const maxLeadBodyBytes = 64 << 10 // 64KB

// LeadHandler принимает заявки клиентов с сайта. Без авторизации — доступен любому клиенту.
// Клиент может только оставить заявку; просмотр и удаление — в админке, см.
// internal/handlers/admin.LeadHandler.
type LeadHandler struct {
	leadService *services.LeadService
	log         *logger.Logger
}

// NewLeadHandler создает новый обработчик заявок.
func NewLeadHandler(leadService *services.LeadService, log *logger.Logger) *LeadHandler {
	return &LeadHandler{
		leadService: leadService,
		log:         log,
	}
}

// CreateLead создает заявку клиента.
// @Summary      Оставить заявку
// @Description  Создает заявку клиента с сайта. Телефон — в международном формате (+79991234567), пробелы/скобки/дефисы допускаются. source по умолчанию — contact_form.
// @Tags         leads
// @Accept       json
// @Produce      json
// @Param        lead  body      models.CreateLeadRequest  true  "Данные заявки"
// @Success      201  {object}  models.Lead
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      429  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/leads [post]
func (h *LeadHandler) CreateLead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxLeadBodyBytes)

	var req models.CreateLeadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	lead, err := h.leadService.Create(req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrSpamDetected):
			// Бот получает обычный успешный ответ и не узнает, что заявка не сохранена.
			httpresponse.WriteJSON(w, http.StatusCreated, struct{}{})
		case errors.Is(err, services.ErrDuplicateSubmission):
			httpresponse.WriteError(w, http.StatusTooManyRequests, httpresponse.TooManyRequestsMessage)
		case errors.Is(err, services.ErrInvalidLead):
			httpresponse.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, repository.ErrServiceNotFound):
			httpresponse.WriteError(w, http.StatusBadRequest, "service not found")
		default:
			h.log.WithError(err).Error("failed to create lead")
			httpresponse.WriteError(w, http.StatusInternalServerError, "failed to create lead")
		}
		return
	}

	httpresponse.WriteJSON(w, http.StatusCreated, lead)
}
