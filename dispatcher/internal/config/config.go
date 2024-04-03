package config

import (
	"fmt"
	"os"
	"time"

	"github.com/llm-operator/job-manager/common/pkg/db"
	"gopkg.in/yaml.v3"
)

// DebugConfig is the debug configuration.
type DebugConfig struct {
	AutoMigrate    bool   `yaml:"autoMigrate"`
	KubeconfigPath string `yaml:"kubeconfigPath"`
}

// Config is the configuration.
type Config struct {
	JobPollingInterval time.Duration `yaml:"jobPollingInterval"`
	JobNamespace       string        `yaml:"jobNamespace"`

	Database db.Config `yaml:"database"`

	Debug DebugConfig `yaml:"debug"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.JobPollingInterval <= 0 {
		return fmt.Errorf("job polling interval must be greater than 0")
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
