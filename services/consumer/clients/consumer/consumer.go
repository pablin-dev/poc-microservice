package consumer

import (
	"context"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	KafkaReceiveTopic = "Income" // TODO: sync with configuration
)

// KafkaConsumerInterface defines the interface for the Kafka consumer functionalities used.
// This allows for mocking the kafka.Reader in tests.
type KafkaConsumerInterface interface {
	ReadMessage(ctx context.Context) (kafka.Message, error)
	Close() error
}

// Consumer holds the Kafka consumer instance (now using the interface)
type Consumer struct {
	reader KafkaConsumerInterface
}

// kafkaConsumerCreator is a function type that allows injecting a mock kafka.NewConsumer for testing.
type kafkaConsumerCreator func(config *kafka.ReaderConfig) (KafkaConsumerInterface, error)

// defaultKafkaConsumerCreator is the default implementation that calls the real kafka.NewConsumer.
var defaultKafkaConsumerCreator kafkaConsumerCreator = func(config *kafka.ReaderConfig) (KafkaConsumerInterface, error) {
	return kafka.NewReader(*config), nil // kafka.NewReader takes a config struct, not a pointer
}

// NewConsumer initializes a new Kafka consumer.
// It uses a kafkaConsumerCreator to allow for dependency injection in tests.
func NewConsumer(bootstrapServers, groupID string) (*Consumer, error) {
	log.Println("Consumer Service: Setting up Kafka consumer...")

	// Default to earliest for auto.offset.reset as per original confluent-kafka-go config
	startOffset := kafka.FirstOffset

	config := kafka.ReaderConfig{
		Brokers:   []string{bootstrapServers},
		GroupID:   groupID,
		Topic:     KafkaReceiveTopic, // Topic is set here for the reader
		MinBytes:  10e3,              // 10KB
		MaxBytes:  10e6,              // 10MB
		MaxWait:   1 * time.Second,   // Maximum amount of time to wait for new data to come when fetching batches.
		StartOffset: startOffset,
	}

	c, err := defaultKafkaConsumerCreator(&config) // Use the creator function
	if err != nil {
		log.Printf("Consumer Service: Failed to create Kafka consumer: %v", err)
		return nil, err
	}
	log.Println("Consumer Service: Kafka consumer created.")
	return &Consumer{reader: c}, nil
}

// ReadMessage reads a message from Kafka
func (c *Consumer) ReadMessage(ctx context.Context) (kafka.Message, error) {
	msg, err := c.reader.ReadMessage(ctx)
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		log.Printf("Consumer Service: Consumer error reading message: %v\n", err)
	}
	return msg, err
}

// Close closes the consumer connection
func (c *Consumer) Close() error {
	log.Println("Consumer Service: Closing Kafka consumer.")
	err := c.reader.Close()
	if err != nil {
		log.Printf("Consumer Service: Error closing Kafka consumer: %v", err)
	}
	log.Println("Consumer Service: Kafka consumer closed.")
	return err
}


