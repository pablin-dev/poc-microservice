package producer

import (
	"context"
	"errors"
	"log"
	"os"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockKafkaProducer is a mock implementation of KafkaProducerInterface for testing.
type MockKafkaProducer struct {
	mock.Mock
}

func (m *MockKafkaProducer) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	args := m.Called(ctx, msgs)
	return args.Error(0)
}

func (m *MockKafkaProducer) Close() error {
	args := m.Called()
	return args.Error(0)
}



// TestNewProducer tests the NewProducer function.
func TestNewProducer(t *testing.T) {
	originalLogOutput := log.Writer()
	log.SetOutput(os.Stdout)
	defer log.SetOutput(originalLogOutput)

	t.Run("Successfully creates producer", func(t *testing.T) {
		mockKafka := new(MockKafkaProducer)
		mockKafka.On("Close").Return(nil).Once()

		// Override the default creator for this test
		oldKafkaProducerCreator := defaultKafkaProducerCreator
		defaultKafkaProducerCreator = func(writer *kafka.Writer) (KafkaProducerInterface, error) {
			return mockKafka, nil
		}
		defer func() { defaultKafkaProducerCreator = oldKafkaProducerCreator }()

		p, err := NewProducer("localhost:9092")

		assert.NoError(t, err)
		assert.NotNil(t, p)
		assert.NotNil(t, p.writer)

		err = p.Close()
		assert.NoError(t, err)
		mockKafka.AssertExpectations(t)
	})

	t.Run("Returns error if Kafka producer creation fails", func(t *testing.T) {
		// Override the default creator to return an error
		oldKafkaProducerCreator := defaultKafkaProducerCreator
		defaultKafkaProducerCreator = func(writer *kafka.Writer) (KafkaProducerInterface, error) {
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
	mockKafka.On("WriteMessages", mock.Anything, mock.AnythingOfType("[]kafka.Message")).Return(nil).Once()
	mockKafka.On("Close").Return(nil).Once()

	// Override the default creator for this test
	oldKafkaProducerCreator := defaultKafkaProducerCreator
	defaultKafkaProducerCreator = func(writer *kafka.Writer) (KafkaProducerInterface, error) {
		return mockKafka, nil
	}
	defer func() { defaultKafkaProducerCreator = oldKafkaProducerCreator }()

	p, err := NewProducer("localhost:9092")
	require.NoError(t, err)

	message := kafka.Message{
		Topic: "test-topic",
		Value: []byte("test-message"),
		Headers: []kafka.Header{{Key: "test", Value: []byte("header")}},
	}

	err = p.Produce(message)

	assert.NoError(t, err)
	mockKafka.AssertCalled(t, "WriteMessages", mock.Anything, []kafka.Message{message}) // Assert with the actual message
	err = p.Close()
	assert.NoError(t, err)
	mockKafka.AssertExpectations(t)
}

func TestProducer_Produce_Error(t *testing.T) {
	originalLogOutput := log.Writer()
	log.SetOutput(os.Stdout)
	defer log.SetOutput(originalLogOutput)

	mockKafka := new(MockKafkaProducer)
	mockKafka.On("WriteMessages", mock.Anything, mock.AnythingOfType("[]kafka.Message")).Return(errors.New("kafka produce error")).Once()
	mockKafka.On("Close").Return(nil).Once()

	oldKafkaProducerCreator := defaultKafkaProducerCreator
	defaultKafkaProducerCreator = func(writer *kafka.Writer) (KafkaProducerInterface, error) {
		return mockKafka, nil
	}
	defer func() { defaultKafkaProducerCreator = oldKafkaProducerCreator }()

	p, err := NewProducer("localhost:9092")
	require.NoError(t, err)

	message := kafka.Message{
		Topic: "test-topic",
		Value: []byte("test-message"),
		Headers: []kafka.Header{},
	}

	err = p.Produce(message)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kafka produce error")
	mockKafka.AssertCalled(t, "WriteMessages", mock.Anything, []kafka.Message{message}) // Assert with the actual message
	err = p.Close()
	assert.NoError(t, err)
	mockKafka.AssertExpectations(t)
}



func TestProducer_Close(t *testing.T) {
	originalLogOutput := log.Writer()
	log.SetOutput(os.Stdout)
	defer log.SetOutput(originalLogOutput)

	mockKafka := new(MockKafkaProducer)
	mockKafka.On("Close").Return(nil).Once()

	oldKafkaProducerCreator := defaultKafkaProducerCreator
	defaultKafkaProducerCreator = func(writer *kafka.Writer) (KafkaProducerInterface, error) {
		return mockKafka, nil
	}
	defer func() { defaultKafkaProducerCreator = oldKafkaProducerCreator }()

	p, err := NewProducer("localhost:9092")
	require.NoError(t, err)

	err = p.Close()
	assert.NoError(t, err)

	mockKafka.AssertCalled(t, "Close")
	mockKafka.AssertExpectations(t)
}
