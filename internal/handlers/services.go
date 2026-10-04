package handlers

import (
	"encoding/json"
	"fmt"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/redis"
	"nexera_back/internal/services"
	"net/http"
	"strconv"
)

// ServiceHandler представляет обработчик услуг
type ServiceHandler struct {
	serviceService *services.ServiceService
	redisClient    *redis.Client
	log            *logger.Logger
}

// NewServiceHandler создает новый обработчик услуг
func NewServiceHandler(
	serviceService *services.ServiceService,
	redisClient *redis.Client,
	log *logger.Logger,
) *ServiceHandler {
	return &ServiceHandler{
		serviceService: serviceService,
		redisClient:    redisClient,
		log:            log,
	}
}

// CreateService создает услугу
// @Summary      Создать услугу
// @Description  Создает новую услугу в каталоге
// @Tags         services
// @Accept       json
// @Produce      json
// @Param        request  body      models.CreateServiceRequest  true  "Данные услуги"
// @Success      201  {object}  models.Service
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/services [post]
func (h *ServiceHandler) CreateService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	var req models.CreateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.validateCreateServiceRequest(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Создание услуги
	service, err := h.serviceService.CreateService(&req)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to create service")
		return
	}

	// Кешируем услугу в Redis
	cacheKey := redis.GenerateKey(redis.KeyPrefixService, strconv.Itoa(service.ID))
	if err := h.redisClient.Set(r.Context(), cacheKey, service, defaultCacheTTL); err != nil {
		h.log.WithError(err).Error("Failed to cache service")
	}

	writeJSONResponse(w, http.StatusCreated, service)
}

// validateCreateServiceRequest валидирует запрос на создание услуги
func (h *ServiceHandler) validateCreateServiceRequest(req *models.CreateServiceRequest) error {
	if req.Name == "" {
		return fmt.Errorf("service name is required")
	}
	return nil
}
