package consumer

import (
	"log"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

const (
	KafkaReceiveTopic = "Income" // TODO: sync with configuration
)

// KafkaConsumerInterface defines the interface for the Kafka consumer functionalities used.
// This allows for mocking the kafka.Consumer in tests.
type KafkaConsumerInterface interface {
	SubscribeTopics(topics []string, rebalanceCb kafka.RebalanceCb) error
	ReadMessage(timeout time.Duration) (*kafka.Message, error)
	Close() error // <--- Change this line
	// CommitMessage is also part of the kafka.Consumer API, but not directly used by this Consumer struct.
	// If it were used, it should be added here.
	CommitMessage(msg *kafka.Message) ([]kafka.TopicPartition, error)
}

// Consumer holds the Kafka consumer instance (now using the interface)
type Consumer struct {
	reader KafkaConsumerInterface
}

// kafkaConsumerCreator is a function type that allows injecting a mock kafka.NewConsumer for testing.
type kafkaConsumerCreator func(conf *kafka.ConfigMap) (KafkaConsumerInterface, error)

// defaultKafkaConsumerCreator is the default implementation that calls the real kafka.NewConsumer.
var defaultKafkaConsumerCreator kafkaConsumerCreator = func(conf *kafka.ConfigMap) (KafkaConsumerInterface, error) {
	return kafka.NewConsumer(conf)
}

// NewConsumer initializes a new Kafka consumer.
// It uses a kafkaConsumerCreator to allow for dependency injection in tests.
func NewConsumer(bootstrapServers, groupID string) (*Consumer, error) {
	log.Println("Consumer Service: Setting up Kafka consumer...")

	conf := &kafka.ConfigMap{
		"bootstrap.servers": bootstrapServers,
		"group.id":          groupID,
		"auto.offset.reset": "earliest",
	}

	c, err := defaultKafkaConsumerCreator(conf) // Use the creator function
	if err != nil {
		log.Printf("Consumer Service: Failed to create Kafka consumer: %v", err)
		return nil, err
	}
	log.Println("Consumer Service: Kafka consumer created.")
	return &Consumer{reader: c}, nil
}

// SubscribeTopics subscribes the consumer to the specified topics
func (c *Consumer) SubscribeTopics(topics []string) error {
	log.Printf("Consumer Service: Subscribing to Kafka topics: %v", topics)
	err := c.reader.SubscribeTopics(topics, nil)
	if err != nil {
		log.Printf("Consumer Service: Failed to subscribe to topics %v: %v", topics, err)
		return err
	}
	log.Printf("Consumer Service: Kafka Consumer subscribed to topics: %v", topics)
	return nil
}

// ReadMessage reads a message from Kafka
func (c *Consumer) ReadMessage(timeout time.Duration) (*kafka.Message, error) {
	msg, err := c.reader.ReadMessage(timeout)
	if err != nil && err.(kafka.Error).Code() != kafka.ErrTimedOut {
		log.Printf("Consumer Service: Consumer error reading message: %v\n", err)
	}
	return msg, err
}

// Close closes the consumer connection
func (c *Consumer) Close() {
	log.Println("Consumer Service: Closing Kafka consumer.")
	c.reader.Close()
	log.Println("Consumer Service: Kafka consumer closed.")
}

// KafkaStringPtr is a helper to get a string pointer for Kafka topic
func KafkaStringPtr(s string) *string { return &s }
