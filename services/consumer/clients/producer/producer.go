package producer

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

const (
	KafkaResponseTopic = "Output" // TODO: sync with configuration
)

// KafkaProducerInterface defines the interface for the Kafka producer functionalities used.
// This allows for mocking the kafka.Writer in tests.
type KafkaProducerInterface interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

// Producer holds the Kafka producer instance (now using the interface)
type Producer struct {
	writer KafkaProducerInterface
}

// kafkaProducerCreator is a function type that allows injecting a mock kafka.NewProducer for testing.
type kafkaProducerCreator func(writer *kafka.Writer) (KafkaProducerInterface, error)

// defaultKafkaProducerCreator is the default implementation that calls the real kafka.NewProducer.
var defaultKafkaProducerCreator kafkaProducerCreator = func(writer *kafka.Writer) (KafkaProducerInterface, error) {
	return writer, nil
}

// NewProducer initializes a new Kafka producer.
// It uses a kafkaProducerCreator to allow for dependency injection in tests.
func NewProducer(bootstrapServers string) (*Producer, error) {
	log.Println("Consumer Service: Setting up Kafka producer...")

	writer := &kafka.Writer{
		Addr:     kafka.TCP(bootstrapServers),
		Balancer: &kafka.LeastBytes{},
	}

	p, err := defaultKafkaProducerCreator(writer) // Use the creator function
	if err != nil {
		log.Printf("Consumer Service: Failed to create Kafka producer: %v", err)
		return nil, err
	}
	log.Println("Consumer Service: Kafka producer created.")

	return &Producer{writer: p}, nil
}

// Produce sends a message to Kafka
func (p *Producer) Produce(message kafka.Message) error {
	log.Printf("Consumer Service: Producing message to Kafka topic: %s", message.Topic)
	err := p.writer.WriteMessages(context.Background(), message)
	if err != nil {
		log.Printf("Consumer Service: Failed to produce message to Kafka: %v", err)
		return err
	}
	log.Println("Consumer Service: Message sent to Kafka producer for delivery.")
	return nil
}



// Close closes the producer connection
func (p *Producer) Close() error {
	log.Println("Consumer Service: Closing Kafka producer.")
	err := p.writer.Close()
	if err != nil {
		log.Printf("Consumer Service: Error closing Kafka producer: %v", err)
	}
	log.Println("Consumer Service: Kafka producer closed.")
	return err
}


