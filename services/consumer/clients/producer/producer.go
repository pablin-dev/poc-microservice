package producer

import (
	"log"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

const (
	KafkaResponseTopic = "Output" // TODO: sync with configuration
)

// KafkaProducerInterface defines the interface for the Kafka producer functionalities used.
// This allows for mocking the kafka.Producer in tests.
type KafkaProducerInterface interface {
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error
	Events() chan kafka.Event
	Flush(timeoutMs int) int
	Close()
}

// Producer holds the Kafka producer instance (now using the interface)
type Producer struct {
	writer KafkaProducerInterface
}

// kafkaProducerCreator is a function type that allows injecting a mock kafka.NewProducer for testing.
type kafkaProducerCreator func(conf *kafka.ConfigMap) (KafkaProducerInterface, error)

// defaultKafkaProducerCreator is the default implementation that calls the real kafka.NewProducer.
var defaultKafkaProducerCreator kafkaProducerCreator = func(conf *kafka.ConfigMap) (KafkaProducerInterface, error) {
	return kafka.NewProducer(conf)
}

// NewProducer initializes a new Kafka producer.
// It uses a kafkaProducerCreator to allow for dependency injection in tests.
func NewProducer(bootstrapServers string) (*Producer, error) {
	log.Println("Consumer Service: Setting up Kafka producer...")

	conf := &kafka.ConfigMap{"bootstrap.servers": bootstrapServers}

	p, err := defaultKafkaProducerCreator(conf) // Use the creator function
	if err != nil {
		log.Printf("Consumer Service: Failed to create Kafka producer: %v", err)
		return nil, err
	}
	log.Println("Consumer Service: Kafka producer created.")

	// Delivery report handler for producer
	go func() {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					log.Printf("Consumer Service: Delivery failed: %v\n", ev.TopicPartition)
				} else {
					log.Printf("Consumer Service: Delivered message to topic %s [%d] at offset %v\n",
						*ev.TopicPartition.Topic, ev.TopicPartition.Partition, ev.TopicPartition.Offset)
				}
			}
		}
	}()

	return &Producer{writer: p}, nil
}

// Produce sends a message to Kafka
func (p *Producer) Produce(topic *string, value []byte, headers []kafka.Header) error {
	log.Printf("Consumer Service: Producing message to Kafka topic: %s", *topic)
	deliveryErr := p.writer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: topic, Partition: kafka.PartitionAny},
		Value:          value,
		Headers:        headers,
	}, nil)
	if deliveryErr != nil {
		log.Printf("Consumer Service: Failed to produce message to Kafka: %v", deliveryErr)
		return deliveryErr
	}
	log.Println("Consumer Service: Message sent to Kafka producer for delivery.")
	return nil
}

// Flush waits for all messages to be delivered
func (p *Producer) Flush(timeoutMs int) int {
	log.Println("Consumer Service: Flushing Kafka producer...")
	remaining := p.writer.Flush(timeoutMs)
	log.Printf("Consumer Service: Producer flushed. Remaining messages: %d", remaining)
	return remaining
}

// Close closes the producer connection
func (p *Producer) Close() {
	log.Println("Consumer Service: Closing Kafka producer.")
	p.writer.Close()
	log.Println("Consumer Service: Kafka producer closed.")
}

// KafkaStringPtr is a helper to get a string pointer for Kafka topic
func KafkaStringPtr(s string) *string { return &s }
