package framework

import (
	"fmt"
	"kafka-soap-e2e-test/tests/clients"
	"time"
)

// Framework holds all clients and configurations needed for E2E tests.
type Framework struct {
	KafkaClient      *clients.KafkaClient
	MountebankClient *clients.MountebankClient
	KycClient        *clients.AdminAPIClient // New field
}

// NewFramework initializes and returns a new Framework instance.
func NewFramework(configPath string) (*Framework, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load framework config: %w", err)
	}

	// Initialize Kafka Client
	kafkaClient, err := clients.NewKafkaClient(cfg.Kafka.BootstrapServers)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka client: %w", err)
	}

	// Initialize Mountebank Client
	mountebankClient := clients.NewMountebankClient(cfg.Mountebank.URL)
	mountebankClient.HTTPClient.Timeout = time.Duration(cfg.Mountebank.TimeoutInSeconds) * time.Second

	// Initialize Admin API Client
	kycClient := clients.NewAdminAPIClient(cfg.KycAdmin.BaseURL)

	return &Framework{
		KafkaClient:      kafkaClient,
		MountebankClient: mountebankClient,
		KycClient:        kycClient,
	}, nil
}
