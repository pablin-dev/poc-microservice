package framework

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v2"
)

var validate *validator.Validate

func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())
}

// KafkaConfig defines the structure for Kafka-specific settings in the config.yaml
type KafkaConfig struct {
	BootstrapServers []string `yaml:"bootstrapServers"`
}

// MountebankConfig defines the structure for Mountebank-specific settings in the config.yaml
type MountebankConfig struct {
	URL              string `yaml:"url"`
	TimeoutInSeconds int    `yaml:"timeoutInSeconds"`
}

// Config defines the overall structure of the config.yaml
type Config struct {
	Kafka      KafkaConfig      `yaml:"kafka"`
	Mountebank MountebankConfig `yaml:"mountebank"`
	KycAdmin   KycAdminConfig   `yaml:"kycAdmin"` // New field
}

// KycAdminConfig defines the structure for KYC Admin API settings in the config.yaml
type KycAdminConfig struct {
	BaseURL string `yaml:"baseURL"`
}

// LoadConfig reads the config.yaml file and unmarshals it into a Config struct.
func LoadConfig(configPath string) (*Config, error) {
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var config Config
	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config data: %w", err)
	}

	// Validate the configuration
	if err := validate.Struct(&config); err != nil {
		panic(fmt.Errorf("configuration validation failed: %w", err))
	}

	return &config, nil
}
