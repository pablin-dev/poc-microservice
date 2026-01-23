package clients

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// KafkaClient holds Kafka producer, consumer, and admin client instances.
type KafkaClient struct {
	Producer *kafka.Producer
	Consumer *kafka.Consumer
	Admin    *kafka.AdminClient
	BootstrapServers string
}

// NewKafkaClient initializes a new KafkaClient.
func NewKafkaClient(bootstrapServers string) (*KafkaClient, error) {
	var err error
	kc := &KafkaClient{
		BootstrapServers: bootstrapServers,
	}

	// Initialize Producer
	kc.Producer, err = kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": bootstrapServers})
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	// Initialize Consumer
	kc.Consumer, err = kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": bootstrapServers,
		"group.id":          "e2e-test-group",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		kc.Producer.Close()
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	// Initialize AdminClient
	kc.Admin, err = kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": bootstrapServers})
	if err != nil {
		kc.Producer.Close()
		kc.Consumer.Close()
		return nil, fmt.Errorf("failed to create Kafka admin client: %w", err)
	}

	return kc, nil
}

// Close closes all Kafka client connections.
func (kc *KafkaClient) Close() {
	if kc.Producer != nil {
		kc.Producer.Close()
	}
	if kc.Consumer != nil {
		kc.Consumer.Close()
	}
	if kc.Admin != nil {
		kc.Admin.Close()
	}
}

// WaitForKafka polls Kafka until it's ready to serve requests or a timeout occurs.
func (kc *KafkaClient) WaitForKafka(timeout time.Duration) error {
	log.Printf("Waiting for Kafka to be ready at %s for %s", kc.BootstrapServers, timeout)
	endTime := time.Now().Add(timeout)
	for time.Now().Before(endTime) {
		// Try to get cluster metadata to confirm active connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := kc.Admin.ClusterID(ctx)
		cancel()

		if err == nil {
			log.Println("Kafka is ready.")
			return nil
		}
		log.Printf("Kafka not yet ready: %v. Retrying...", err)
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("timed out waiting for Kafka to be ready")
}

// CreateKafkaTopic creates a Kafka topic if it doesn't already exist.
func (kc *KafkaClient) CreateKafkaTopic(topic string, numPartitions int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	topicSpec := kafka.TopicSpecification{
		Topic:             topic,
		NumPartitions:     numPartitions,
		ReplicationFactor: 3, // Changed for 3-broker setup
	}

	results, _ := kc.Admin.CreateTopics(ctx, []kafka.TopicSpecification{topicSpec}) // Ignore the top-level error

	for _, result := range results {
		log.Printf("Topic: %s, Error Code: %d, Error String: %s", result.Topic, result.Error.Code(), result.Error.String())
		if result.Error.Code() != kafka.ErrNoError && result.Error.Code() != kafka.ErrTopicAlreadyExists {
			return fmt.Errorf("failed to create topic %s: %w", result.Topic, result.Error)
		}
		if result.Error.Code() == kafka.ErrTopicAlreadyExists {
			log.Printf("Topic '%s' already exists, skipping creation.", topic)
		}
	}
	log.Printf("Topic '%s' created successfully or already exists.", topic)
	return nil
}

// DeleteKafkaTopic deletes a Kafka topic.
func (kc *KafkaClient) DeleteKafkaTopic(topic string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := kc.Admin.DeleteTopics(ctx, []string{topic})
	if err != nil {
		return fmt.Errorf("failed to delete topic %s: %w", topic, err)
	}

	for _, result := range results {
		log.Printf("Topic: %s, Error Code: %d, Error String: %s", result.Topic, result.Error.Code(), result.Error.String())
		if result.Error.Code() != kafka.ErrNoError && result.Error.Code() != kafka.ErrUnknownTopicOrPart {
			return fmt.Errorf("failed to delete topic %s: %w", topic, result.Error)
		}
		if result.Error.Code() == kafka.ErrUnknownTopicOrPart {
			log.Printf("Topic '%s' not found, skipping deletion.", topic)
		}
	}
	log.Printf("Topic '%s' deleted successfully or did not exist.", topic)
	return nil
}

// KafkaStringPtr is a helper to get a string pointer for Kafka topic
func KafkaStringPtr(s string) *string { return &s }
