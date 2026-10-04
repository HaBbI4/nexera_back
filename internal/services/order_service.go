package services

import (
	"database/sql"
	"fmt"
	"nexera_back/internal/database"
	"nexera_back/internal/kafka"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"time"
)

// OrderService представляет сервис для работы с заказами
type OrderService struct {
	db       *database.DB
	log      *logger.Logger
	producer *kafka.Producer
}

// NewOrderService создает новый сервис для работы с заказами.
// producer может быть nil (например, в тестах) — публикация событий в этом случае пропускается.
func NewOrderService(
	db *database.DB,
	log *logger.Logger,
	producer *kafka.Producer,
) *OrderService {
	return &OrderService{
		db:       db,
		log:      log,
		producer: producer,
	}
}

// GetOrders возвращает список заказов вместе с автомобилем, клиентом и услугами
func (svc *OrderService) GetOrders(limit, offset int) ([]*models.Order, error) {
	query := `
		SELECT id, customer_id, car_id, total_amount,
			scheduled_at, started_at, completed_at,
			status, comment, created_at, updated_at
		FROM orders
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
		return nil, fmt.Errorf("error getting orders: %w", err)
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		order := &models.Order{}
		if err := rows.Scan(
			&order.ID, &order.CustomerID, &order.CarID, &order.TotalAmount,
			&order.ScheduledAt, &order.StartedAt, &order.CompletedAt,
			&order.Status, &order.Comment,
			&order.CreatedAt, &order.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	for _, order := range orders {
		if err := svc.attachRelations(order); err != nil {
			return nil, err
		}
	}

	return orders, nil
}

// GetActiveOrders возвращает заказы, которые еще не завершены и не отменены
// (used by telegram-bot для команды /orders)
func (svc *OrderService) GetActiveOrders(limit, offset int) ([]*models.Order, error) {
	query := `
		SELECT id, customer_id, car_id, total_amount,
			scheduled_at, started_at, completed_at,
			status, comment, created_at, updated_at
		FROM orders
		WHERE status NOT IN ($1, $2)
		ORDER BY scheduled_at ASC
	`
	args := []interface{}{models.OrderStatusCompleted, models.OrderStatusCancelled}
	argIndex := 3

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
		return nil, fmt.Errorf("error getting active orders: %w", err)
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		order := &models.Order{}
		if err := rows.Scan(
			&order.ID, &order.CustomerID, &order.CarID, &order.TotalAmount,
			&order.ScheduledAt, &order.StartedAt, &order.CompletedAt,
			&order.Status, &order.Comment,
			&order.CreatedAt, &order.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	for _, order := range orders {
		if err := svc.attachRelations(order); err != nil {
			return nil, err
		}
	}

	return orders, nil
}

// GetOrder возвращает заказ по ID вместе с автомобилем, клиентом и услугами
func (svc *OrderService) GetOrder(orderID int) (*models.Order, error) {
	order := &models.Order{}

	query := `
		SELECT id, customer_id, car_id, total_amount,
			scheduled_at, started_at, completed_at,
			status, comment, created_at, updated_at
		FROM orders
		WHERE id = $1
	`

	err := svc.db.QueryRow(query, orderID).Scan(
		&order.ID, &order.CustomerID, &order.CarID, &order.TotalAmount,
		&order.ScheduledAt, &order.StartedAt, &order.CompletedAt,
		&order.Status, &order.Comment,
		&order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("error getting order: %w", err)
	}

	if err := svc.attachRelations(order); err != nil {
		return nil, err
	}

	return order, nil
}

// CreateOrder создает заказ
func (svc *OrderService) CreateOrder(req *models.CreateOrderRequest) (*models.Order, error) {
	// Запускаем транзакцию
	tx, err := svc.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Расчет общей суммы
	var totalAmount float64
	for _, item := range req.Items {
		totalAmount += item.Price * float64(item.Quantity)
	}

	// Создание заказа
	order := &models.Order{
		CustomerID:  req.CustomerID,
		CarID:       req.CarID,
		ScheduledAt: req.ScheduledAt,
		Status:      models.OrderStatusCreated,
		TotalAmount: totalAmount,
		Comment:     req.Comment,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO orders (customer_id, car_id, 
		                    scheduled_at, total_amount, 
		                    status, comment, 
		                    created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	err = tx.QueryRow(
		query,
		order.CustomerID,
		order.CarID,
		order.ScheduledAt,
		order.TotalAmount,
		order.Status,
		order.Comment,
		order.CreatedAt,
		order.UpdatedAt,
	).Scan(&order.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Добавление услуг в заказ
	for _, item := range req.Items {
		serviceID, serviceName, err := svc.resolveServiceItem(tx, &item)
		if err != nil {
			return nil, err
		}

		itemQuery := `
			INSERT INTO order_items (order_id, service_id, service_name, quantity, price, comment)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`

		var itemID int
		if err = tx.QueryRow(
			itemQuery,
			order.ID,
			serviceID,
			serviceName,
			item.Quantity,
			item.Price,
			item.Comment,
		).Scan(&itemID); err != nil {
			return nil, fmt.Errorf("failed to create order item: %w", err)
		}

		order.Items = append(order.Items, models.OrderItem{
			ID:          itemID,
			OrderID:     order.ID,
			ServiceID:   serviceID,
			ServiceName: serviceName,
			Quantity:    item.Quantity,
			Price:       item.Price,
			Comment:     item.Comment,
		})
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	svc.log.WithFields(map[string]interface{}{
		"order_id":     order.ID,
		"customer_id":  order.CustomerID,
		"total_amount": order.TotalAmount,
	}).Info("Order created successfully")

	// Заказ уже создан и закоммичен — ошибки ниже не должны провалить запрос.
	if car, err := svc.getCar(order.CarID); err == nil {
		order.Car = car
	} else {
		svc.log.WithError(err).WithField("order_id", order.ID).Warn("Failed to load car for order.created event")
	}
	if customer, err := svc.getCustomer(order.CustomerID); err == nil {
		order.Customer = customer
	} else {
		svc.log.WithError(err).WithField("order_id", order.ID).Warn("Failed to load customer for order.created event")
	}

	if svc.producer != nil {
		if err := svc.producer.PublishOrderCreated(order); err != nil {
			svc.log.WithError(err).WithField("order_id", order.ID).Error("Failed to publish order.created event")
		}
	}

	return order, nil
}

// resolveServiceItem возвращает id и имя услуги для order_item. Если передан ServiceID,
// используется существующая услуга. Если ServiceID не передан, а передан ServiceName —
// сначала создается новая услуга в таблице services в рамках той же транзакции.
func (svc *OrderService) resolveServiceItem(tx *sql.Tx, item *models.CreateOrderItemRequest) (int, string, error) {
	if item.ServiceID > 0 {
		var serviceName string
		if err := tx.QueryRow("SELECT name FROM services WHERE id = $1", item.ServiceID).Scan(&serviceName); err != nil {
			if err == sql.ErrNoRows {
				return 0, "", fmt.Errorf("service %d not found", item.ServiceID)
			}
			return 0, "", fmt.Errorf("failed to get service %d: %w", item.ServiceID, err)
		}
		return item.ServiceID, serviceName, nil
	}

	now := time.Now()
	query := `
		INSERT INTO services (name, description, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var serviceID int
	if err := tx.QueryRow(
		query,
		item.ServiceName,
		item.ServiceDescription,
		item.ServiceIsActive,
		now, now,
	).Scan(&serviceID); err != nil {
		return 0, "", fmt.Errorf("failed to create service %q: %w", item.ServiceName, err)
	}

	return serviceID, item.ServiceName, nil
}

// attachRelations подгружает автомобиль, клиента и услуги для заказа
func (svc *OrderService) attachRelations(order *models.Order) error {
	car, err := svc.getCar(order.CarID)
	if err != nil {
		return err
	}
	order.Car = car

	customer, err := svc.getCustomer(order.CustomerID)
	if err != nil {
		return err
	}
	order.Customer = customer

	items, err := svc.getOrderItems(order.ID)
	if err != nil {
		return err
	}
	order.Items = items

	return nil
}

// getCar подзапрос получения автомобиля заказов
func (svc *OrderService) getCar(carID int) (*models.Car, error) {
	car := &models.Car{}

	query := `
		SELECT id, customer_id, brand, model, year, license_plate,
			vin, comment, created_at, updated_at
		FROM cars
		WHERE id = $1
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

// getCustomer подзапрос получения пользователя заказов
func (svc *OrderService) getCustomer(customerID int) (*models.Customer, error) {
	customer := &models.Customer{}

	query := `
		SELECT id, name, phone, email, comment, created_at, updated_at
		FROM customers
		WHERE id = $1
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

// getOrderItems подзапрос получения услуг заказов
func (svc *OrderService) getOrderItems(orderID int) ([]models.OrderItem, error) {
	query := `
		SELECT id, order_id, service_id, service_name, quantity, price, comment, created_at
		FROM order_items
		WHERE order_id = $1
		ORDER BY id
	`

	rows, err := svc.db.Query(query, orderID)
	if err != nil {
		return nil, fmt.Errorf("error getting order items: %w", err)
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		if err := rows.Scan(
			&item.ID, &item.OrderID, &item.ServiceID, &item.ServiceName,
			&item.Quantity, &item.Price, &item.Comment, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan order item: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}
