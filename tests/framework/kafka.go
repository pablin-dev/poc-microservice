package framework

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaClient holds Kafka producer, consumer, and admin client instances.
type KafkaClient struct {
	Producer         *kafka.Writer
	Consumer         *kafka.Reader
	Admin            *kafka.Client // For admin operations like topic creation/deletion
	BootstrapServers []string
	Context          context.Context
	Cancel           context.CancelFunc
}

// NewKafkaClient initializes a new KafkaClient.
func NewKafkaClient(bootstrapServers []string, consumerGroup, consumerOffset, consumerTopic string) (*KafkaClient, error) {
	ctx, cancel := context.WithCancel(context.Background())
	bootstrapServersStr := strings.Join(bootstrapServers, ",")

	kc := &KafkaClient{
		BootstrapServers: bootstrapServers,
		Context:          ctx,
		Cancel:           cancel,
	}

	// Initialize Producer
	kc.Producer = &kafka.Writer{
		Addr:     kafka.TCP(bootstrapServersStr),
		Balancer: &kafka.LeastBytes{},
	}

	// Determine StartOffset based on consumerOffset string
	var startOffset int64
	switch consumerOffset {
	case "earliest":
		startOffset = kafka.FirstOffset
	case "latest":
		startOffset = kafka.LastOffset
	default:
		// Default to latest or handle specific offset parsing if needed
		startOffset = kafka.LastOffset
		log.Printf("Unknown consumer offset setting '%s', defaulting to 'latest'.", consumerOffset)
	}

	// Initialize Consumer
	kc.Consumer = kafka.NewReader(kafka.ReaderConfig{
		Brokers:   bootstrapServers,
		GroupID:   consumerGroup,
		Topic:     consumerTopic, // Use the provided consumerTopic
		MinBytes:  10e3,              // 10KB
		MaxBytes:  10e6,              // 10MB
		MaxWait:   1 * time.Second,
		StartOffset: startOffset, // Set the determined start offset
	})

	// Initialize Admin Client (using kafka.Client for admin operations)
	kc.Admin = &kafka.Client{
		Addr:    kafka.TCP(bootstrapServersStr),
		Timeout: 5 * time.Second,
	}

	return kc, nil
}

// Close closes all Kafka client connections.
func (kc *KafkaClient) Close() {
	if kc.Producer != nil {
		if err := kc.Producer.Close(); err != nil {
			log.Printf("Error closing Kafka producer: %v", err)
		}
	}
	if kc.Consumer != nil {
		if err := kc.Consumer.Close(); err != nil {
			log.Printf("Error closing Kafka consumer: %v", err)
		}
	}
	// kafka.Client does not have a Close method. Its resources are tied to the context.
	if kc.Cancel != nil {
		kc.Cancel()
	}
}

// WaitForKafka polls Kafka until it's ready to serve requests or a timeout occurs.
func (kc *KafkaClient) WaitForKafka(timeout time.Duration) error {
	log.Printf("Waiting for Kafka to be ready at %s for %s", strings.Join(kc.BootstrapServers, ","), timeout)
	endTime := time.Now().Add(timeout)
	for time.Now().Before(endTime) {
		ctx, cancel := context.WithTimeout(kc.Context, 5*time.Second)
		defer cancel() // Ensure context is cancelled even if check succeeds

		conn, err := kafka.DialContext(ctx, "tcp", kc.BootstrapServers[0]) // Dial to the first bootstrap server
		if err != nil {
			log.Printf("Kafka not yet ready (Dial failed): %v. Retrying...", err)
			time.Sleep(1 * time.Second)
			continue
		}
		defer conn.Close() // Close the connection after each check

		// Optionally, check controller to ensure it's healthy
		_, err = conn.Controller()
		if err != nil {
			log.Printf("Kafka not yet ready (Controller failed): %v. Retrying...", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// Read partitions to verify cluster metadata can be fetched
		_, err = conn.ReadPartitions()
		if err == nil {
			log.Println("Kafka is ready.")
			return nil
		}

		log.Printf("Kafka not yet ready (ReadPartitions failed): %v. Retrying...", err)
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("timed out waiting for Kafka to be ready")
}

// CreateKafkaTopic creates a Kafka topic if it doesn't already exist.
func (kc *KafkaClient) CreateKafkaTopic(topic string, numPartitions int) error {
	ctx, cancel := context.WithTimeout(kc.Context, 10*time.Second)
	defer cancel()

	createRequest := kafka.CreateTopicsRequest{
		Topics: []kafka.TopicConfig{
			{
				Topic:             topic,
				NumPartitions:     numPartitions,
				ReplicationFactor: 1, // Changed for E2E test to simplify startup
			},
		},
		ValidateOnly: false,
	}

	createResponse, err := kc.Admin.CreateTopics(ctx, &createRequest)
	if err != nil {
		return fmt.Errorf("failed to send CreateTopics request for topic %s: %w", topic, err)
	}

	// Check individual topic errors in the response
	if createResponseError, ok := createResponse.Errors[topic]; ok && createResponseError != nil {
		if errors.Is(createResponseError, kafka.TopicAlreadyExists) {
			log.Printf("Topic '%s' already exists, skipping creation.", topic)
			return nil
		}
		return fmt.Errorf("failed to create topic %s: %w", topic, createResponseError)
	}

	log.Printf("Topic '%s' created successfully.", topic)
	return nil
}

// DeleteKafkaTopic deletes a Kafka topic.
func (kc *KafkaClient) DeleteKafkaTopic(topic string) error {
	ctx, cancel := context.WithTimeout(kc.Context, 10*time.Second)
	defer cancel()

	deleteRequest := kafka.DeleteTopicsRequest{
		Topics: []string{topic},
		// You can add IgnoreIfNotFound: true if you want to silently ignore non-existent topics
	}

	deleteResponse, err := kc.Admin.DeleteTopics(ctx, &deleteRequest)
	if err != nil {
		return fmt.Errorf("failed to send DeleteTopics request for topic %s: %w", topic, err)
	}

	// Check individual topic errors in the response
	if deleteResponseError, ok := deleteResponse.Errors[topic]; ok && deleteResponseError != nil {
		if errors.Is(deleteResponseError, kafka.UnknownTopicOrPartition) {
			log.Printf("Topic '%s' not found, skipping deletion.", topic)
			return nil
		}
		return fmt.Errorf("failed to delete topic %s: %w", topic, deleteResponseError)
	}

	log.Printf("Topic '%s' deleted successfully.", topic)
	return nil
}

// CleanQueue resets the consumer group offset to the latest available message for all partitions of a given topic.
func (kc *KafkaClient) CleanQueue(topic, consumerGroup string) error {
	log.Printf("Cleaning queue for topic '%s' for consumer group '%s'...", topic, consumerGroup)
	ctx, cancel := context.WithTimeout(kc.Context, 10*time.Second)
	defer cancel()

	// 1. Get partition metadata for the topic
	metadata, err := kc.Admin.Metadata(ctx, &kafka.MetadataRequest{Topics: []string{topic}})
	if err != nil {
		return fmt.Errorf("failed to get metadata for topic %s: %w", topic, err)
	}

	if len(metadata.Topics) == 0 {
		return fmt.Errorf("topic %s not found in metadata", topic)
	}

	topicMetadata := metadata.Topics[0]
	if topicMetadata.Error != nil {
		return fmt.Errorf("error in metadata for topic %s: %w", topic, topicMetadata.Error)
	}

	offsetsToCommit := make([]kafka.OffsetCommit, len(topicMetadata.Partitions))
	for i, partition := range topicMetadata.Partitions {
		// 2. For each partition, get the latest offset
		listOffsetsResp, err := kc.Admin.ListOffsets(ctx, &kafka.ListOffsetsRequest{
			Topics: map[string][]kafka.OffsetRequest{
				topic: {
					kafka.LastOffsetOf(partition.ID), // Request latest offset
				},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to list offsets for topic %s partition %d: %w", topic, partition.ID, err)
		}

		partitionOffsets, ok := listOffsetsResp.Topics[topic]
		if !ok || len(partitionOffsets) == 0 {
			return fmt.Errorf("no offsets found for topic %s partition %d", topic, partition.ID)
		}

		latestOffset := partitionOffsets[0].LastOffset

		// 3. Prepare offset commit for the latest offset
		offsetsToCommit[i] = kafka.OffsetCommit{
			Partition: partition.ID,
			Offset:    latestOffset,
			Metadata:  "reset by test framework",
		}
	}

	// 4. Commit these latest offsets for the consumer group
	_, err = kc.Admin.OffsetCommit(ctx, &kafka.OffsetCommitRequest{
		GroupID: consumerGroup,
		Topics: map[string][]kafka.OffsetCommit{
			topic: offsetsToCommit,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit offsets for topic %s consumer group %s: %w", topic, consumerGroup, err)
	}

	log.Printf("Successfully cleaned queue for topic '%s' for consumer group '%s'.", topic, consumerGroup)
	return nil
}

// ProduceMessage produces a message to the specified Kafka topic.
func (kc *KafkaClient) ProduceMessage(message kafka.Message) error {
	err := kc.Producer.WriteMessages(kc.Context, message)
	if err != nil {
		return fmt.Errorf("failed to produce message to topic %s: %w", message.Topic, err)
	}

	log.Printf("Delivered message to topic %s\n", message.Topic)
	return nil
}

// ConsumeMessage consumes a single message from the specified Kafka topic within a given timeout.
func (kc *KafkaClient) ConsumeMessage(timeout time.Duration) (kafka.Message, error) {

	ctx, cancel := context.WithTimeout(kc.Context, timeout)
	defer cancel()

	msg, err := kc.Consumer.ReadMessage(ctx)
	if err != nil {
		return kafka.Message{}, fmt.Errorf("failed to consume message: %w", err)
	}

	log.Printf("Consumed message from topic %s at offset %v\n", msg.Topic, msg.Offset)
	return msg, nil
}


