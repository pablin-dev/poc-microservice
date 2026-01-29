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
	Producer         *kafka.Producer
	Consumer         *kafka.Consumer
	Admin            *kafka.AdminClient
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
		// Try to describe the cluster to confirm active connection and cluster metadata availability
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := kc.Admin.DescribeCluster(ctx)
		cancel()

		if err == nil {
			log.Println("Kafka is ready.")
			return nil
		}
		log.Printf("Kafka not yet ready (DescribeCluster failed): %v. Retrying...", err)
		time.Sleep(5 * time.Second)
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
		ReplicationFactor: 1, // Changed for E2E test to simplify startup
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

// ProduceMessage produces a message to the specified Kafka topic.
func (kc *KafkaClient) ProduceMessage(topic string, key, value []byte) error {
	deliveryChan := make(chan kafka.Event)
	defer close(deliveryChan)

	err := kc.Producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Key:            key,
		Value:          value,
	}, deliveryChan)
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	e := <-deliveryChan
	m := e.(*kafka.Message)

	if m.TopicPartition.Error != nil {
		return fmt.Errorf("delivery failed: %v", m.TopicPartition.Error)
	}
	log.Printf("Delivered message to topic %s [%d] at offset %v\n",
		*m.TopicPartition.Topic, m.TopicPartition.Partition, m.TopicPartition.Offset)
	return nil
}

// ConsumeMessage consumes a single message from the specified Kafka topic within a given timeout.
func (kc *KafkaClient) ConsumeMessage(topic string, timeout time.Duration) (*kafka.Message, error) {
	err := kc.Consumer.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to topic %s: %w", topic, err)
	}

	// Use a context with timeout for polling
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out after %s waiting for message from topic %s", timeout, topic)
		default:
			ev := kc.Consumer.Poll(100) // Poll with a small timeout in milliseconds
			if ev == nil {
				continue // No event received, continue polling
			}

			switch e := ev.(type) {
			case *kafka.Message:
				log.Printf("Consumed message from topic %s [%d] at offset %v\n",
					*e.TopicPartition.Topic, e.TopicPartition.Partition, e.TopicPartition.Offset)
				return e, nil
			case kafka.Error:
				return nil, fmt.Errorf("kafka consumer error: %v", e)
			case kafka.PartitionEOF:
				// End of partition, can happen in some scenarios, continue polling
				continue
			default:
				// Ignore other events like stats
				continue
			}
		}
	}
}

// KafkaStringPtr is a helper to get a string pointer for Kafka topic
func KafkaStringPtr(s string) *string { return &s }
