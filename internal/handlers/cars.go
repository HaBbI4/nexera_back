package handlers

import (
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/redis"
	"nexera_back/internal/services"
	"net/http"
	"strconv"
	"strings"
)

// CarHandler представляет обработчик автомобилей
type CarHandler struct {
	carService  *services.CarService
	redisClient *redis.Client
	log         *logger.Logger
}

// NewCarHandler создает новый обработчик автомобилей
func NewCarHandler(
	carService *services.CarService,
	redisClient *redis.Client,
	log *logger.Logger,
) *CarHandler {
	return &CarHandler{
		carService:  carService,
		redisClient: redisClient,
		log:         log,
	}
}

// GetCars получает список автомобилей
// @Summary      Список автомобилей
// @Description  Возвращает список автомобилей с пагинацией
// @Tags         cars
// @Produce      json
// @Param        limit   query     int  false  "Количество записей (по умолчанию 50, максимум 100)"
// @Param        offset  query     int  false  "Смещение (по умолчанию 0)"
// @Success      200  {array}   models.Car
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/cars [get]
func (h *CarHandler) GetCars(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	query := r.URL.Query()

	// Парсинг параметров
	limit := 50 // По умолчанию
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	cars, err := h.carService.GetCars(limit, offset)
	if err != nil {
		h.log.WithError(err).Error("Error getting cars")
		writeErrorResponse(w, http.StatusInternalServerError, "Error getting cars")
		return
	}

	writeJSONResponse(w, http.StatusOK, cars)
}

// GetCar получается автомобиль по ID
// @Summary      Получить автомобиль
// @Description  Возвращает автомобиль по ID
// @Tags         cars
// @Produce      json
// @Param        id   path      int  true  "ID автомобиля"
// @Success      200  {object}  models.Car
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/cars/{id} [get]
func (h *CarHandler) GetCar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	carID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || carID <= 0 {
		writeErrorResponse(w, http.StatusBadRequest, "invalid car id")
		return
	}

	// Попытка получить из кеша
	cacheKey := redis.GenerateKey(redis.KeyPrefixCar, strconv.Itoa(carID))
	var car models.Car
	if err := h.redisClient.Get(r.Context(), cacheKey, &car); err == nil {
		h.log.WithField("car_id", carID).Debug("Car retrieved from cache")
		writeJSONResponse(w, http.StatusOK, &car)
		return
	}

	// Получение из баз данных
	carPtr, err := h.carService.GetCar(carID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeErrorResponse(w, http.StatusNotFound, "Car not found")
		} else {
			h.log.WithError(err).Error("Failed to get car")
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to get car")
		}
		return
	}

	// Кэширование автомобиля
	if err := h.redisClient.Set(r.Context(), cacheKey, carPtr, defaultCacheTTL); err != nil {
		h.log.WithError(err).Error("Failed to set car")
	}

	writeJSONResponse(w, http.StatusOK, carPtr)
}
