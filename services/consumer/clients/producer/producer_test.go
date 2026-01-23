package producer

import (
	"errors"
	"log"
	"os"
	"testing"
	"time"


	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockKafkaProducer is a mock implementation of KafkaProducerInterface for testing.
type MockKafkaProducer struct {
	mock.Mock
	EventsChannel chan kafka.Event
}

func (m *MockKafkaProducer) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	args := m.Called(msg, deliveryChan)
	return args.Error(0)
}

func (m *MockKafkaProducer) Events() chan kafka.Event {
	args := m.Called()
	return args.Get(0).(chan kafka.Event)
}

func (m *MockKafkaProducer) Flush(timeoutMs int) int {
	args := m.Called(timeoutMs)
	return args.Int(0)
}

func (m *MockKafkaProducer) Close() {
	m.Called()
}

// TestKafkaStringPtr tests the helper function KafkaStringPtr.
func TestKafkaStringPtr(t *testing.T) {
	testString := "testTopic"
	ptr := KafkaStringPtr(testString)

	assert.NotNil(t, ptr)
	assert.Equal(t, testString, *ptr)
}

// TestNewProducer tests the NewProducer function.
func TestNewProducer(t *testing.T) {
	originalLogOutput := log.Writer()
	log.SetOutput(os.Stdout)
	defer log.SetOutput(originalLogOutput)

	t.Run("Successfully creates producer", func(t *testing.T) {
		mockKafka := new(MockKafkaProducer)
		eventsChan := make(chan kafka.Event) // Create a channel for Events()
		mockKafka.On("Events").Return(eventsChan) // Mock the Events channel for the goroutine
		mockKafka.On("Close").Return().Once() // Expect Close() to be called

		// Override the default creator for this test
		oldKafkaProducerCreator := defaultKafkaProducerCreator
		defaultKafkaProducerCreator = func(conf *kafka.ConfigMap) (KafkaProducerInterface, error) {
			return mockKafka, nil
		}
		defer func() { defaultKafkaProducerCreator = oldKafkaProducerCreator }()

		p, err := NewProducer("localhost:9092")

		assert.NoError(t, err)
		assert.NotNil(t, p)
		assert.NotNil(t, p.writer)

		// Close the Events channel after a short delay to allow the goroutine to exit gracefully
		// This prevents the goroutine from potentially holding up AssertExpectations
		go func() {
			time.Sleep(10 * time.Millisecond) // Give the goroutine a moment to start
			close(eventsChan)
		}()

		p.Close() // Ensure close is called on the mock
		mockKafka.AssertExpectations(t)
	})

	t.Run("Returns error if Kafka producer creation fails", func(t *testing.T) {
		// Override the default creator to return an error
		oldKafkaProducerCreator := defaultKafkaProducerCreator
		defaultKafkaProducerCreator = func(conf *kafka.ConfigMap) (KafkaProducerInterface, error) {
			return nil, errors.New("failed to create Kafka producer mock")
		}
		defer func() { defaultKafkaProducerCreator = oldKafkaProducerCreator }()

		p, err := NewProducer("invalid-bootstrap-server")

		assert.Error(t, err)
		assert.Nil(t, p)
		assert.Contains(t, err.Error(), "failed to create Kafka producer mock")
	})
}

func TestProducer_Produce(t *testing.T) {
	originalLogOutput := log.Writer()
	log.SetOutput(os.Stdout)
	defer log.SetOutput(originalLogOutput)

	mockKafka := new(MockKafkaProducer)
	eventsChan := make(chan kafka.Event)
	mockKafka.On("Produce", mock.Anything, mock.Anything).Return(nil).Once()
	mockKafka.On("Events").Return(eventsChan).Maybe()
	mockKafka.On("Close").Return().Once()

	defer close(eventsChan) // Ensure the events channel is closed

	// Inject the mock into the Producer struct
	// This approach is needed if NewProducer is not used for specific method tests
	// but for this refactored producer.go, it's better to use NewProducer for setup.
	oldKafkaProducerCreator := defaultKafkaProducerCreator
	defaultKafkaProducerCreator = func(conf *kafka.ConfigMap) (KafkaProducerInterface, error) {
		return mockKafka, nil
	}
	defer func() { defaultKafkaProducerCreator = oldKafkaProducerCreator }()

	p, err := NewProducer("localhost:9092")
	require.NoError(t, err)
	
	topic := KafkaStringPtr("test-topic")
	value := []byte("test-message")
	headers := []kafka.Header{{Key: "test", Value: []byte("header")}}

	err = p.Produce(topic, value, headers)

	assert.NoError(t, err)
	mockKafka.AssertCalled(t, "Produce", mock.Anything, mock.Anything)
	p.Close()
	mockKafka.AssertExpectations(t)
}

func TestProducer_Produce_Error(t *testing.T) {
	originalLogOutput := log.Writer()
	log.SetOutput(os.Stdout)
	defer log.SetOutput(originalLogOutput)

	mockKafka := new(MockKafkaProducer)
	eventsChan := make(chan kafka.Event)
	mockKafka.On("Produce", mock.Anything, mock.Anything).Return(errors.New("kafka produce error")).Once()
	mockKafka.On("Events").Return(eventsChan).Maybe()
	mockKafka.On("Close").Return().Once()

	defer close(eventsChan) // Ensure the events channel is closed

	oldKafkaProducerCreator := defaultKafkaProducerCreator
	defaultKafkaProducerCreator = func(conf *kafka.ConfigMap) (KafkaProducerInterface, error) {
		return mockKafka, nil
	}
	defer func() { defaultKafkaProducerCreator = oldKafkaProducerCreator }()

	p, err := NewProducer("localhost:9092")
	require.NoError(t, err)

	topic := KafkaStringPtr("test-topic")
	value := []byte("test-message")
	headers := []kafka.Header{}

	err = p.Produce(topic, value, headers)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kafka produce error")
	mockKafka.AssertCalled(t, "Produce", mock.Anything, mock.Anything)
	p.Close()
	mockKafka.AssertExpectations(t)
}

func TestProducer_Flush(t *testing.T) {
	originalLogOutput := log.Writer()
	log.SetOutput(os.Stdout)
	defer log.SetOutput(originalLogOutput)

	mockKafka := new(MockKafkaProducer)
	eventsChan := make(chan kafka.Event)
	mockKafka.On("Flush", 1000).Return(0).Once()
	mockKafka.On("Events").Return(eventsChan).Maybe()
	mockKafka.On("Close").Return().Once()

	defer close(eventsChan) // Ensure the events channel is closed

	oldKafkaProducerCreator := defaultKafkaProducerCreator
	defaultKafkaProducerCreator = func(conf *kafka.ConfigMap) (KafkaProducerInterface, error) {
		return mockKafka, nil
	}
	defer func() { defaultKafkaProducerCreator = oldKafkaProducerCreator }()

	p, err := NewProducer("localhost:9092")
	require.NoError(t, err)

	remaining := p.Flush(1000)

	assert.Equal(t, 0, remaining)
	mockKafka.AssertCalled(t, "Flush", 1000)
	p.Close()
	mockKafka.AssertExpectations(t)
}

func TestProducer_Close(t *testing.T) {
	originalLogOutput := log.Writer()
	log.SetOutput(os.Stdout)
	defer log.SetOutput(originalLogOutput)

	mockKafka := new(MockKafkaProducer)
	eventsChan := make(chan kafka.Event)
	mockKafka.On("Close").Return().Once()
	mockKafka.On("Events").Return(eventsChan).Maybe() // Events must be mocked for NewProducer goroutine

	defer close(eventsChan) // Ensure the events channel is closed

	oldKafkaProducerCreator := defaultKafkaProducerCreator
	defaultKafkaProducerCreator = func(conf *kafka.ConfigMap) (KafkaProducerInterface, error) {
		return mockKafka, nil
	}
	defer func() { defaultKafkaProducerCreator = oldKafkaProducerCreator }()

	p, err := NewProducer("localhost:9092")
	require.NoError(t, err)

	p.Close()

	mockKafka.AssertCalled(t, "Close")
	mockKafka.AssertExpectations(t)
}