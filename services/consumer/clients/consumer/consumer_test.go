package consumer

import (
	"context"
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockKafkaConsumer is a mock implementation of KafkaConsumerInterface for testing.
type MockKafkaConsumer struct {
	mock.Mock
}

func (m *MockKafkaConsumer) ReadMessage(ctx context.Context) (kafka.Message, error) {
	args := m.Called(ctx)
	msg := args.Get(0)
	if msg == nil { // Handle case where nil kafka.Message is returned
		return kafka.Message{}, args.Error(1)
	}
	return msg.(kafka.Message), args.Error(1)
}

func (m *MockKafkaConsumer) Close() error {
	args := m.Called()
	return args.Error(0)
}



// TestNewConsumer tests the NewConsumer function.
func TestNewConsumer(t *testing.T) {
	originalLogOutput := log.Writer()
	log.SetOutput(os.Stdout)
	defer log.SetOutput(originalLogOutput)

	t.Run("Successfully creates consumer", func(t *testing.T) {
		mockKafka := new(MockKafkaConsumer)
		// No Close call expected directly after NewConsumer
		// mockKafka.On("Close").Return(nil).Once()

		oldKafkaConsumerCreator := defaultKafkaConsumerCreator
		defaultKafkaConsumerCreator = func(config *kafka.ReaderConfig) (KafkaConsumerInterface, error) {
			return mockKafka, nil
		}
		defer func() { defaultKafkaConsumerCreator = oldKafkaConsumerCreator }()

		c, err := NewConsumer("localhost:9092", "test-group")

		assert.NoError(t, err)
		assert.NotNil(t, c)
		assert.NotNil(t, c.reader)
		// No direct Close call here either for the test
		// c.Close()
		mockKafka.AssertExpectations(t)
	})

	t.Run("Returns error if Kafka consumer creation fails", func(t *testing.T) {
		oldKafkaConsumerCreator := defaultKafkaConsumerCreator
		defaultKafkaConsumerCreator = func(config *kafka.ReaderConfig) (KafkaConsumerInterface, error) {
			return nil, errors.New("failed to create Kafka consumer mock")
		}
		defer func() { defaultKafkaConsumerCreator = oldKafkaConsumerCreator }()

		c, err := NewConsumer("invalid-bootstrap-server", "test-group")

		assert.Error(t, err)
		assert.Nil(t, c)
		assert.Contains(t, err.Error(), "failed to create Kafka consumer mock")
	})
}



func TestConsumer_ReadMessage(t *testing.T) {
	originalLogOutput := log.Writer()
	log.SetOutput(os.Stdout)
	defer log.SetOutput(originalLogOutput)

	t.Run("Successfully reads message", func(t *testing.T) {
		mockKafka := new(MockKafkaConsumer)
		testTopic := "test-topic"
		testMessage := kafka.Message{
			Topic: testTopic,
			Partition: 0, Offset: 1,
			Value: []byte("hello"),
		}
		mockCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		mockKafka.On("ReadMessage", mock.AnythingOfType("context.Context")).Return(testMessage, nil).Once()
		mockKafka.On("Close").Return(nil).Once()

		oldKafkaConsumerCreator := defaultKafkaConsumerCreator
		defaultKafkaConsumerCreator = func(config *kafka.ReaderConfig) (KafkaConsumerInterface, error) {
			return mockKafka, nil
		}
		defer func() { defaultKafkaConsumerCreator = oldKafkaConsumerCreator }()

		c, err := NewConsumer("localhost:9092", "test-group")
		require.NoError(t, err)

		msg, err := c.ReadMessage(mockCtx)

		assert.NoError(t, err)
		assert.Equal(t, testMessage, msg)
		mockKafka.AssertCalled(t, "ReadMessage", mock.AnythingOfType("context.Context"))
		c.Close()
		mockKafka.AssertExpectations(t)
	})

	t.Run("ReadMessage returns timeout error", func(t *testing.T) {
		mockKafka := new(MockKafkaConsumer)
		timeoutErr := context.DeadlineExceeded // Use context.DeadlineExceeded for timeout
		mockCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		mockKafka.On("ReadMessage", mock.AnythingOfType("context.Context")).Return(kafka.Message{}, timeoutErr).Once()
		mockKafka.On("Close").Return(nil).Once()

		oldKafkaConsumerCreator := defaultKafkaConsumerCreator
		defaultKafkaConsumerCreator = func(config *kafka.ReaderConfig) (KafkaConsumerInterface, error) {
			return mockKafka, nil
		}
		defer func() { defaultKafkaConsumerCreator = oldKafkaConsumerCreator }()

		c, err := NewConsumer("localhost:9092", "test-group")
		require.NoError(t, err)

		msg, err := c.ReadMessage(mockCtx)

		assert.Error(t, err)
		assert.True(t, errors.Is(err, timeoutErr)) // Use errors.Is for context errors
		assert.Equal(t, kafka.Message{}, msg)
		mockKafka.AssertCalled(t, "ReadMessage", mock.AnythingOfType("context.Context"))
		c.Close()
		mockKafka.AssertExpectations(t)
	})

	t.Run("ReadMessage returns other error", func(t *testing.T) {
		mockKafka := new(MockKafkaConsumer)
		otherErr := errors.New("some other kafka error") // Generic error
		mockCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		mockKafka.On("ReadMessage", mock.AnythingOfType("context.Context")).Return(kafka.Message{}, otherErr).Once()
		mockKafka.On("Close").Return(nil).Once()

		oldKafkaConsumerCreator := defaultKafkaConsumerCreator
		defaultKafkaConsumerCreator = func(config *kafka.ReaderConfig) (KafkaConsumerInterface, error) {
			return mockKafka, nil
		}
		defer func() { defaultKafkaConsumerCreator = oldKafkaConsumerCreator }()

		c, err := NewConsumer("localhost:9092", "test-group")
		require.NoError(t, err)

		msg, err := c.ReadMessage(mockCtx)

		assert.Error(t, err)
		assert.True(t, errors.Is(err, otherErr)) // Use errors.Is
		assert.Equal(t, kafka.Message{}, msg)
		mockKafka.AssertCalled(t, "ReadMessage", mock.AnythingOfType("context.Context"))
		c.Close()
		mockKafka.AssertExpectations(t)
	})
}

func TestConsumer_Close(t *testing.T) {
	originalLogOutput := log.Writer()
	log.SetOutput(os.Stdout)
	defer log.SetOutput(originalLogOutput)

	mockKafka := new(MockKafkaConsumer)
	mockKafka.On("Close").Return(nil).Once()

	oldKafkaConsumerCreator := defaultKafkaConsumerCreator
	defaultKafkaConsumerCreator = func(config *kafka.ReaderConfig) (KafkaConsumerInterface, error) {
		return mockKafka, nil
	}
	defer func() { defaultKafkaConsumerCreator = oldKafkaConsumerCreator }()

	c, err := NewConsumer("localhost:9092", "test-group")
	require.NoError(t, err)

	err = c.Close() // Call Close and capture its error
	assert.NoError(t, err)

	mockKafka.AssertCalled(t, "Close")
	mockKafka.AssertExpectations(t)
}
