package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the configuration.
type Config struct {
	HTTPPort int `yaml:"httpPort"`

	Proxies []ProxyConfig `yaml:"proxies"`
}

// ProxyConfig is the configuration for a proxy.
type ProxyConfig struct {
	Name string `yaml:"name"`

	BaseURL string `yaml:"baseUrl"`

	AuthToken string `yaml:"authToken"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.HTTPPort <= 0 {
		return fmt.Errorf("httpPort must be greater than 0")
	}
	if len(c.Proxies) == 0 {
		return fmt.Errorf("proxies must not be empty")
	}
	for i, p := range c.Proxies {
		if err := p.Validate(); err != nil {
			return fmt.Errorf("proxies[%d]: %s", i, err)
		}
	}
	return nil
}

// Validate validates the configuration.
func (c *ProxyConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name must not be empty")
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
