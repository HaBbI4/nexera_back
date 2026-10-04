package services

import (
	"database/sql"
	"fmt"
	"nexera_back/internal/database"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"time"
)

// CustomerService представляет сервис для работы с клиентами
type CustomerService struct {
	db  *database.DB
	log *logger.Logger
}

// NewCustomerService создает новый сервис для работы с клиентами
func NewCustomerService(db *database.DB, log *logger.Logger) *CustomerService {
	return &CustomerService{
		db:  db,
		log: log,
	}
}

// GetCustomers возвращает клиентов
func (svc *CustomerService) GetCustomers(limit, offset int) ([]*models.Customer, error) {
	query := `
		SELECT id, name, phone, email, comment, created_at, updated_at
		FROM customers
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
		return nil, fmt.Errorf("error getting customer: %w", err)
	}

	defer rows.Close()

	var customers []*models.Customer
	for rows.Next() {
		customer := &models.Customer{}
		if err := rows.Scan(
			&customer.ID, &customer.Name,
			&customer.Phone, &customer.Email,
			&customer.Comment,
			&customer.CreatedAt, &customer.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error scan customer: %w", err)
		}
		customers = append(customers, customer)
	}

	return customers, nil
}

// GetCustomer возвращает клиента по ID
func (svc *CustomerService) GetCustomer(customerID int) (*models.Customer, error) {
	customer := &models.Customer{}

	query := `
		SELECT id, name, phone, email, comment, created_at, updated_at
		FROM customers
		WHERE id=$1
	`

	err := svc.db.QueryRow(query, customerID).Scan(
		&customer.ID, &customer.Name,
		&customer.Phone, &customer.Email,
		&customer.Comment,
		&customer.CreatedAt, &customer.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("customer not found")
		}
		return nil, fmt.Errorf("error getting customer: %w", err)
	}

	return customer, nil
}

// CreateCustomer создает клиента
func (svc *CustomerService) CreateCustomer(req *models.CreateCustomerRequest) (*models.Customer, error) {
	customer := &models.Customer{
		Name:      req.Name,
		Phone:     req.Phone,
		Email:     req.Email,
		Comment:   req.Comment,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	query := `
		INSERT INTO customers (name, phone, email, comment, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	err := svc.db.QueryRow(
		query,
		customer.Name, customer.Phone,
		customer.Email, customer.Comment,
		customer.CreatedAt, customer.UpdatedAt,
	).Scan(&customer.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	svc.log.WithFields(map[string]interface{}{
		"customer_id":    customer.ID,
		"customer_name":  customer.Name,
		"customer_phone": customer.Phone,
		"customer_email": customer.Email,
	}).Info("created customer")

	return customer, nil
}
