package kafka

import (
	"encoding/json"
	"fmt"
	"time"

	"nexera_back/internal/config"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"

	"github.com/IBM/sarama"
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

// Publish публикует событие в топик событий (KAFKA_TOPIC_EVENTS).
func (p *Producer) Publish(event models.Event) error {
	topic := p.topics.Events

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
