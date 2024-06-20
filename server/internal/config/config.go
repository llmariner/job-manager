package config

import (
	"fmt"
	"os"

	"github.com/llm-operator/common/pkg/db"
	"gopkg.in/yaml.v3"
)

// Config is the configuration.
type Config struct {
	GRPCPort              int `yaml:"grpcPort"`
	WorkerServiceGRPCPort int `yaml:"workerServiceGrpcPort"`
	HTTPPort              int `yaml:"httpPort"`

	FileManagerServerAddr        string `yaml:"fileManagerServerAddr"`
	ModelManagerServerAddr       string `yaml:"modelManagerServerAddr"`
	SessionManagerServerEndpoint string `yaml:"sessionManagerServerEndpoint"`

	Database db.Config `yaml:"database"`

	AuthConfig AuthConfig `yaml:"auth"`

	NotebookConfig NotebookConfig `yaml:"notebook"`
}

// NotebookConfig is the notebook configuration.
type NotebookConfig struct {
	ImageTypes map[string]string `yaml:"imageTypes"`
}

// Validate validates the configuration.
func (c *NotebookConfig) Validate() error {
	if len(c.ImageTypes) == 0 {
		return fmt.Errorf("imageTypes must be set")
	}
	return nil
}

// AuthConfig is the authentication configuration.
type AuthConfig struct {
	Enable                 bool   `yaml:"enable"`
	RBACInternalServerAddr string `yaml:"rbacInternalServerAddr"`
}

// Validate validates the configuration.
func (c *AuthConfig) Validate() error {
	if !c.Enable {
		return nil
	}
	if c.RBACInternalServerAddr == "" {
		return fmt.Errorf("rbacInternalServerAddr must be set")
	}
	return nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.GRPCPort <= 0 {
		return fmt.Errorf("grpcPort must be greater than 0")
	}
	if c.WorkerServiceGRPCPort <= 0 {
		return fmt.Errorf("workerServiceGRPCPort must be greater than 0")
	}
	if c.HTTPPort <= 0 {
		return fmt.Errorf("httpPort must be greater than 0")
	}
	if c.FileManagerServerAddr == "" {
		return fmt.Errorf("file manager address must be set")
	}
	if c.ModelManagerServerAddr == "" {
		return fmt.Errorf("model manager server address must be set")
	}
	if c.SessionManagerServerEndpoint == "" {
		return fmt.Errorf("session manager server endpoint must be set")
	}
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database: %s", err)
	}
	if err := c.AuthConfig.Validate(); err != nil {
		return fmt.Errorf("auth: %s", err)
	}
	if err := c.NotebookConfig.Validate(); err != nil {
		return fmt.Errorf("notebook: %s", err)
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
