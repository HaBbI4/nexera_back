package services

import (
	"database/sql"
	"fmt"
	"nexera_back/internal/database"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
)

// CarService представляет сервис для работы с автомобилями
type CarService struct {
	db  *database.DB
	log *logger.Logger
}

// NewCarService создает новый сервис для работы с автомобилями
func NewCarService(db *database.DB, log *logger.Logger) *CarService {
	return &CarService{
		db:  db,
		log: log,
	}
}

// GetCars возвращает список всех автомобилей
func (svc *CarService) GetCars(limit, offset int) ([]*models.Car, error) {
	query := `
		SELECT id, customer_id, brand, model, year, license_plate,
			vin, comment, created_at, updated_at
		FROM cars
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
		argIndex++
	}

	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, offset)
	}

	rows, err := svc.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error getting cars: %w", err)
	}

	defer rows.Close()

	var cars []*models.Car
	for rows.Next() {
		car := &models.Car{}
		if err := rows.Scan(
			&car.ID, &car.CustomerID,
			&car.Brand, &car.Model,
			&car.Year, &car.LicensePlate,
			&car.VIN, &car.Comment,
			&car.CreatedAt, &car.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan cars: %w", err)
		}
		cars = append(cars, car)
	}

	return cars, nil
}

// GetCar возвращает автомобиль по ID
func (svc *CarService) GetCar(carID int) (*models.Car, error) {
	car := &models.Car{}

	query := `
		SELECT id, customer_id, brand, model, year, license_plate,
			vin, comment, created_at, updated_at
		FROM cars
		WHERE id=$1
	`

	err := svc.db.QueryRow(query, carID).Scan(
		&car.ID, &car.CustomerID,
		&car.Brand, &car.Model,
		&car.Year, &car.LicensePlate,
		&car.VIN, &car.Comment,
		&car.CreatedAt, &car.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("car not found")
		}
		return nil, fmt.Errorf("error getting car: %w", err)
	}

	return car, nil
}
