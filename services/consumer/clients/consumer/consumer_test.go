package consumer

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

// MockKafkaConsumer is a mock implementation of KafkaConsumerInterface for testing.
type MockKafkaConsumer struct {
	mock.Mock
}

func (m *MockKafkaConsumer) SubscribeTopics(topics []string, rebalanceCb kafka.RebalanceCb) error {
	args := m.Called(topics, rebalanceCb)
	return args.Error(0)
}

func (m *MockKafkaConsumer) ReadMessage(timeout time.Duration) (*kafka.Message, error) {
	args := m.Called(timeout)
	msg := args.Get(0)
	if msg == nil {
		return nil, args.Error(1)
	}
	return msg.(*kafka.Message), args.Error(1)
}

func (m *MockKafkaConsumer) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockKafkaConsumer) CommitMessage(msg *kafka.Message) ([]kafka.TopicPartition, error) {
	args := m.Called(msg)
	return args.Get(0).([]kafka.TopicPartition), args.Error(1)
}

// isNilRebalanceCb is a custom matcher to assert that rebalanceCb is nil.
func isNilRebalanceCb() interface{} {
	return mock.MatchedBy(func(rebalanceCb kafka.RebalanceCb) bool {
		return rebalanceCb == nil
	})
}

// TestKafkaStringPtr tests the helper function KafkaStringPtr.
func TestKafkaStringPtr(t *testing.T) {
	testString := "testTopic"
	ptr := KafkaStringPtr(testString)

	assert.NotNil(t, ptr)
	assert.Equal(t, testString, *ptr)
}

// TestNewConsumer tests the NewConsumer function.
func TestNewConsumer(t *testing.T) {
	originalLogOutput := log.Writer()
	log.SetOutput(os.Stdout)
	defer log.SetOutput(originalLogOutput)

	t.Run("Successfully creates consumer", func(t *testing.T) {
		mockKafka := new(MockKafkaConsumer)
		mockKafka.On("Close").Return(nil).Once()

		oldKafkaConsumerCreator := defaultKafkaConsumerCreator
		defaultKafkaConsumerCreator = func(conf *kafka.ConfigMap) (KafkaConsumerInterface, error) {
			return mockKafka, nil
		}
		defer func() { defaultKafkaConsumerCreator = oldKafkaConsumerCreator }()

		c, err := NewConsumer("localhost:9092", "test-group")

		assert.NoError(t, err)
		assert.NotNil(t, c)
		assert.NotNil(t, c.reader)
		c.Close() // Ensure Close is called on the mock
		mockKafka.AssertExpectations(t)
	})

	t.Run("Returns error if Kafka consumer creation fails", func(t *testing.T) {
		oldKafkaConsumerCreator := defaultKafkaConsumerCreator
		defaultKafkaConsumerCreator = func(conf *kafka.ConfigMap) (KafkaConsumerInterface, error) {
			return nil, errors.New("failed to create Kafka consumer mock")
		}
		defer func() { defaultKafkaConsumerCreator = oldKafkaConsumerCreator }()

		c, err := NewConsumer("invalid-bootstrap-server", "test-group")

		assert.Error(t, err)
		assert.Nil(t, c)
		assert.Contains(t, err.Error(), "failed to create Kafka consumer mock")
	})
}

func TestConsumer_SubscribeTopics(t *testing.T) {
	originalLogOutput := log.Writer()
	log.SetOutput(os.Stdout)
	defer log.SetOutput(originalLogOutput)

	mockKafka := new(MockKafkaConsumer)
	testTopics := []string{"topic1", "topic2"}
	mockKafka.On("SubscribeTopics", testTopics, isNilRebalanceCb()).Return(nil)
	mockKafka.On("Close").Return(nil).Once() // Expected during defer if NewConsumer is used

	oldKafkaConsumerCreator := defaultKafkaConsumerCreator
	defaultKafkaConsumerCreator = func(conf *kafka.ConfigMap) (KafkaConsumerInterface, error) {
		return mockKafka, nil
	}
	defer func() { defaultKafkaConsumerCreator = oldKafkaConsumerCreator }()

	c, err := NewConsumer("localhost:9092", "test-group")
	require.NoError(t, err)

	subscribeErr := c.SubscribeTopics(testTopics)
	assert.NoError(t, subscribeErr)

	mockKafka.AssertCalled(t, "SubscribeTopics", testTopics, isNilRebalanceCb())
	c.Close()
	mockKafka.AssertExpectations(t)
}

func TestConsumer_SubscribeTopics_Error(t *testing.T) {
	originalLogOutput := log.Writer()
	log.SetOutput(os.Stdout)
	defer log.SetOutput(originalLogOutput)

	mockKafka := new(MockKafkaConsumer)
	testTopics := []string{"topic1", "topic2"}
	mockKafka.On("SubscribeTopics", testTopics, isNilRebalanceCb()).Return(errors.New("kafka subscription error"))
	mockKafka.On("Close").Return(nil).Once()

	oldKafkaConsumerCreator := defaultKafkaConsumerCreator
	defaultKafkaConsumerCreator = func(conf *kafka.ConfigMap) (KafkaConsumerInterface, error) {
		return mockKafka, nil
	}
	defer func() { defaultKafkaConsumerCreator = oldKafkaConsumerCreator }()

	c, err := NewConsumer("localhost:9092", "test-group")
	require.NoError(t, err)

	subscribeErr := c.SubscribeTopics(testTopics)

	assert.Error(t, subscribeErr)
	assert.Contains(t, subscribeErr.Error(), "kafka subscription error")
	mockKafka.AssertCalled(t, "SubscribeTopics", testTopics, isNilRebalanceCb())
	c.Close()
	mockKafka.AssertExpectations(t)
}

func TestConsumer_ReadMessage(t *testing.T) {
	originalLogOutput := log.Writer()
	log.SetOutput(os.Stdout)
	defer log.SetOutput(originalLogOutput)

	t.Run("Successfully reads message", func(t *testing.T) {
		mockKafka := new(MockKafkaConsumer)
		testTopic := "test-topic"
		testMessage := &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &testTopic, Partition: 0, Offset: 1},
			Value:          []byte("hello"),
		}
		mockKafka.On("ReadMessage", 100*time.Millisecond).Return(testMessage, nil).Once()
		mockKafka.On("Close").Return(nil).Once()

		oldKafkaConsumerCreator := defaultKafkaConsumerCreator
		defaultKafkaConsumerCreator = func(conf *kafka.ConfigMap) (KafkaConsumerInterface, error) {
			return mockKafka, nil
		}
		defer func() { defaultKafkaConsumerCreator = oldKafkaConsumerCreator }()

		c, err := NewConsumer("localhost:9092", "test-group")
		require.NoError(t, err)

		msg, err := c.ReadMessage(100 * time.Millisecond)

		assert.NoError(t, err)
		assert.Equal(t, testMessage, msg)
		mockKafka.AssertCalled(t, "ReadMessage", 100*time.Millisecond)
		c.Close()
		mockKafka.AssertExpectations(t)
	})

	t.Run("ReadMessage returns timeout error", func(t *testing.T) {
		mockKafka := new(MockKafkaConsumer)
		timeoutErr := kafka.NewError(kafka.ErrTimedOut, "timed out", false)
		mockKafka.On("ReadMessage", 100*time.Millisecond).Return(nil, timeoutErr).Once()
		mockKafka.On("Close").Return(nil).Once()

		oldKafkaConsumerCreator := defaultKafkaConsumerCreator
		defaultKafkaConsumerCreator = func(conf *kafka.ConfigMap) (KafkaConsumerInterface, error) {
			return mockKafka, nil
		}
		defer func() { defaultKafkaConsumerCreator = oldKafkaConsumerCreator }()

		c, err := NewConsumer("localhost:9092", "test-group")
		require.NoError(t, err)

		msg, err := c.ReadMessage(100 * time.Millisecond)

		assert.Error(t, err)
		assert.Nil(t, msg)
		assert.Equal(t, timeoutErr, err)
		mockKafka.AssertCalled(t, "ReadMessage", 100*time.Millisecond)
		c.Close()
		mockKafka.AssertExpectations(t)
	})

	t.Run("ReadMessage returns other error", func(t *testing.T) {
		mockKafka := new(MockKafkaConsumer)
		otherErr := kafka.NewError(kafka.ErrMsgSizeTooLarge, "too large", false)
		mockKafka.On("ReadMessage", 100*time.Millisecond).Return(nil, otherErr).Once()
		mockKafka.On("Close").Return(nil).Once()

		oldKafkaConsumerCreator := defaultKafkaConsumerCreator
		defaultKafkaConsumerCreator = func(conf *kafka.ConfigMap) (KafkaConsumerInterface, error) {
			return mockKafka, nil
		}
		defer func() { defaultKafkaConsumerCreator = oldKafkaConsumerCreator }()

		c, err := NewConsumer("localhost:9092", "test-group")
		require.NoError(t, err)

		msg, err := c.ReadMessage(100 * time.Millisecond)

		assert.Error(t, err)
		assert.Nil(t, msg)
		assert.Equal(t, otherErr, err)
		mockKafka.AssertCalled(t, "ReadMessage", 100*time.Millisecond)
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
	defaultKafkaConsumerCreator = func(conf *kafka.ConfigMap) (KafkaConsumerInterface, error) {
		return mockKafka, nil
	}
	defer func() { defaultKafkaConsumerCreator = oldKafkaConsumerCreator }()

	c, err := NewConsumer("localhost:9092", "test-group")
	require.NoError(t, err)

	c.Close()

	mockKafka.AssertCalled(t, "Close")
	mockKafka.AssertExpectations(t)
}
