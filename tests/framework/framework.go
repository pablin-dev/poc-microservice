package framework

import (
	"fmt"
	"time"
)

// Framework holds all clients and configurations needed for E2E tests.
type Framework struct {
	Config           *Config
	KafkaClient      *KafkaClient
	MountebankClient *MountebankClient
	KycClient        *AdminAPIClient // New field
}

// NewFramework initializes and returns a new Framework instance.
func NewFramework(configPath string) (*Framework, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load framework config: %w", err)
	}

	// Initialize Kafka Client
	kafkaClient, err := NewKafkaClient(cfg.Kafka.BootstrapServers, cfg.Kafka.Consumer.Group, cfg.Kafka.Consumer.Offset, cfg.Kafka.Producer.Topic)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka client: %w", err)
	}

	// Initialize Mountebank Client
	mountebankClient := NewMountebankClient(cfg.Mountebank)
	mountebankClient.HTTPClient.Timeout = time.Duration(cfg.Mountebank.TimeoutInSeconds) * time.Second
	// Initialize Admin API Client
	kycClient := NewAdminAPIClient(cfg.KycAdmin.BaseURL)

	return &Framework{
		KafkaClient:      kafkaClient,
		MountebankClient: mountebankClient,
		KycClient:        kycClient,
		Config:           cfg,
	}, nil
}
