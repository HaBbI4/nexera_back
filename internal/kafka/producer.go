package kafka

import (
	"encoding/json"
	"fmt"
	"time"

	"nexera_back/internal/config"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
)

// Producer представляет Kafka producer
type Producer struct {
	producer sarama.SyncProducer
	log      *logger.Logger
	topics   *config.Topics
}

// NewProducer создает новый Kafka producer
func NewProducer(cfg *config.KafkaConfig, log *logger.Logger) (*Producer, error) {
	saramaCfg := sarama.NewConfig()
	saramaCfg.Producer.RequiredAcks = sarama.WaitForAll       // Ждем подтверждения от всех реплик
	saramaCfg.Producer.Retry.Max = 3                          // Максимум 3 попытки
	saramaCfg.Producer.Return.Successes = true                // Возвращаем успешные результаты
	saramaCfg.Producer.Compression = sarama.CompressionSnappy // Сжатие данных

	producer, err := sarama.NewSyncProducer(cfg.Brokers, saramaCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	log.Info("Kafka producer created successfully")

	return &Producer{
		producer: producer,
		log:      log,
		topics:   &cfg.Topics,
	}, nil
}

// Close закрывает producer
func (p *Producer) Close() error {
	return p.producer.Close()
}

// PublishOrderCreated публикует событие создания заказа
func (p *Producer) PublishOrderCreated(order *models.Order) error {
	var customerName, carBrand, carModel string
	if order.Customer != nil {
		customerName = order.Customer.Name
	}
	if order.Car != nil {
		carBrand = order.Car.Brand
		carModel = order.Car.Model
	}

	event := models.Event{
		ID:        uuid.New(),
		Type:      models.EventTypeOrderCreated,
		Timestamp: time.Now(),
		Data: models.OrderCreatedEvent{
			OrderID:      order.ID,
			CustomerName: customerName,
			CarBrand:     carBrand,
			CarModel:     carModel,
			TotalAmount:  order.TotalAmount,
			CreatedAt:    order.CreatedAt,
		},
	}

	return p.publishEvent(p.topics.Orders, event)
}

// PublishOrderStatusChanged публикует событие изменения статуса заказа
func (p *Producer) PublishOrderStatusChanged(orderID int, oldStatus, newStatus models.OrderStatus) error {
	event := models.Event{
		ID:        uuid.New(),
		Type:      models.EventTypeOrderStatusChanged,
		Timestamp: time.Now(),
		Data: models.OrderStatusChangedEvent{
			OrderID:   orderID,
			OldStatus: oldStatus,
			NewStatus: newStatus,
			Timestamp: time.Now(),
		},
	}

	return p.publishEvent(p.topics.Orders, event)
}

// publishEvent публикует событие в указанный топик
func (p *Producer) publishEvent(topic string, event models.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	message := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(event.ID.String()),
		Value: sarama.ByteEncoder(data),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("event_type"),
				Value: []byte(event.Type),
			},
			{
				Key:   []byte("timestamp"),
				Value: []byte(event.Timestamp.Format(time.RFC3339)),
			},
		},
	}

	partition, offset, err := p.producer.SendMessage(message)
	if err != nil {
		return fmt.Errorf("failed to send message to topic %s: %w", topic, err)
	}

	p.log.WithField("topic", topic).
		WithField("partition", partition).
		WithField("offset", offset).
		WithField("event_type", event.Type).
		WithField("event_id", event.ID).
		Debug("Event published successfully")

	return nil
}
