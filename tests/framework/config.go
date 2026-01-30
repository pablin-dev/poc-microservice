package framework

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// MountebankPorts defines the structure for Mountebank ports
type MountebankPorts struct {
	Admin     int              `koanf:"admin"`
	Imposters []map[string]int `koanf:"imposters"`
}

// MountebankConfig defines the structure for Mountebank-specific settings in the config.yaml
type MountebankConfig struct {
	URL              string          `koanf:"url"`
	Ports            MountebankPorts `koanf:"ports"`
	TimeoutInSeconds int             `koanf:"timeoutInSeconds"`
}

// KafkaConfig defines the structure for Kafka-specific settings in the config.yaml

type KafkaConsumerConfig struct {
	Group  string `koanf:"group"`
	Topic  string `koanf:"topic"`
	Offset string `koanf:"offset"`
}

type KafkaProducerConfig struct {
	Topic string `koanf:"topic"`
}

type KafkaConfig struct {
	BootstrapServers []string            `koanf:"bootstrapServers"`
	Consumer         KafkaConsumerConfig `koanf:"consumer"`
	Producer         KafkaProducerConfig `koanf:"producer"`
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
	k := koanf.New(".")
	validate := validator.New(validator.WithRequiredStructEnabled())

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
