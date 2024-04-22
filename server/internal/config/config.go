package config

import (
	"fmt"
	"os"

	"github.com/llm-operator/job-manager/common/pkg/db"
	"gopkg.in/yaml.v3"
)

// Config is the configuration.
type Config struct {
	GRPCPort int `yaml:"grpcPort"`
	HTTPPort int `yaml:"httpPort"`

	FileManagerServerAddr          string `yaml:"fileManagerServerAddr"`
	ModelManagerInternalServerAddr string `yaml:"modelManagerInternalServerAddr"`

	Database db.Config `yaml:"database"`

	JobNamespace string `yaml:"jobNamespace"`

	Debug DebugConfig `yaml:"debug"`

	AuthConfig AuthConfig `yaml:"auth"`
}

type AuthConfig struct {
	OIDCIssuerURL string `yaml:"oidcIssuerURL"`
	OIDCClientID  string `yaml:"oidcClientID"`
}

// DebugConfig is the debug configuration.
type DebugConfig struct {
	KubeconfigPath string `yaml:"kubeconfigPath"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.GRPCPort <= 0 {
		return fmt.Errorf("grpcPort must be greater than 0")
	}
	if c.HTTPPort <= 0 {
		return fmt.Errorf("httpPort must be greater than 0")
	}
	if c.FileManagerServerAddr == "" {
		return fmt.Errorf("file manager address must be set")
	}
	if c.ModelManagerInternalServerAddr == "" {
		return fmt.Errorf("model manager internal server address must be set")
	}
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database: %s", err)
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
