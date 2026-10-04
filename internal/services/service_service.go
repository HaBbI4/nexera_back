package services

import (
	"fmt"
	"nexera_back/internal/database"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"time"
)

// ServiceService представляет сервис для работы с услугами
type ServiceService struct {
	db  *database.DB
	log *logger.Logger
}

// NewServiceService создает новый сервис для рабты с услугами
func NewServiceService(
	db *database.DB,
	log *logger.Logger,
) *ServiceService {
	return &ServiceService{
		db:  db,
		log: log,
	}
}

// CreateService создает услугу
func (svc *ServiceService) CreateService(req *models.CreateServiceRequest) (*models.Service, error) {
	service := &models.Service{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    req.IsActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO services (name, description, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	err := svc.db.QueryRow(
		query,
		service.Name, service.Description,
		service.IsActive,
		service.CreatedAt, service.UpdatedAt,
	).Scan(&service.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	svc.log.WithFields(map[string]interface{}{
		"service_id":        service.ID,
		"service_name":      service.Name,
		"service_is_active": service.IsActive,
	}).Info("created new service")

	return service, nil
}
