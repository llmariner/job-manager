package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the configuration.
type Config struct {
	HTTPPort int `yaml:"httpPort"`

	BaseURL string `yaml:"baseUrl"`

	AuthToken string `yaml:"authToken"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.HTTPPort <= 0 {
		return fmt.Errorf("httpPort must be greater than 0")
	}
	if c.BaseURL == "" {
		return fmt.Errorf("baseUrl must not be empty")
	}
	if c.AuthToken == "" {
		return fmt.Errorf("authToken must not be empty")
	}

	return nil
}

// Parse parses the configuration file at the given path, returning a new
// Config struct.
func Parse(path string) (Config, error) {
	var config Config

	b, err := os.ReadFile(path)
	if err != nil {
		return config, fmt.Errorf("config: read: %s", err)
	}

	if err = yaml.Unmarshal(b, &config); err != nil {
		return config, fmt.Errorf("config: unmarshal: %s", err)
	}
	return config, nil
}
