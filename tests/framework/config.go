package framework

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

var validate *validator.Validate

func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())
}

// KafkaConfig defines the structure for Kafka-specific settings in the config.yaml
type KafkaConfig struct {
	BootstrapServers []string `koanf:"bootstrapServers"`
}

// MountebankConfig defines the structure for Mountebank-specific settings in the config.yaml
type MountebankConfig struct {
	URL              string `koanf:"url"`
	TimeoutInSeconds int    `koanf:"timeoutInSeconds"`
}

// Config defines the overall structure of the config.yaml
type Config struct {
	Kafka      KafkaConfig      `koanf:"kafka"`
	Mountebank MountebankConfig `koanf:"mountebank"`
	KycAdmin   KycAdminConfig   `koanf:"kycAdmin"` // New field
}

// KycAdminConfig defines the structure for KYC Admin API settings in the config.yaml
type KycAdminConfig struct {
	BaseURL string `koanf:"baseURL"`
}

// LoadConfig reads the config.yaml file and unmarshals it into a Config struct.
func LoadConfig(configPath string) (*Config, error) {
	var k = koanf.New(".")

	// Load YAML config.
	if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("error loading config: %w", err)
	}

	var config Config
	if err := k.Unmarshal("", &config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate the configuration
	if err := validate.Struct(&config); err != nil {
		panic(fmt.Errorf("configuration validation failed: %w", err))
	}

	return &config, nil
}
