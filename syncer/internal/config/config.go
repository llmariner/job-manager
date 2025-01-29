package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the configuration.
type Config struct {
	JobManagerServerSyncerServiceAddr string `yaml:"jobManagerServerSyncerServiceAddr"`
	SessionManagerEndpoint            string `yaml:"sessionManagerEndpoint"`

	KubernetesManager KubernetesManagerConfig `yaml:"kubernetesManager"`
}

// KubernetesManagerConfig is the Kubernetes manager configuration.
type KubernetesManagerConfig struct {
	EnableLeaderElection bool   `yaml:"enableLeaderElection"`
	LeaderElectionID     string `yaml:"leaderElectionID"`

	MetricsBindAddress string `yaml:"metricsBindAddress"`
	HealthBindAddress  string `yaml:"healthBindAddress"`
	PprofBindAddress   string `yaml:"pprofBindAddress"`
}

func (c *KubernetesManagerConfig) validate() error {
	if c.EnableLeaderElection && c.LeaderElectionID == "" {
		return fmt.Errorf("leader election ID must be set")
	}
	return nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.JobManagerServerSyncerServiceAddr == "" {
		return fmt.Errorf("jobManagerServerSyncerServiceAddr must be set")
	}
	if c.SessionManagerEndpoint == "" {
		return fmt.Errorf("sessionManagerEndpoint must be set")
	}
	if err := c.KubernetesManager.validate(); err != nil {
		return fmt.Errorf("kubernetesManager: %s", err)
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
