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
	KubeconfigPath string `yaml:"kubeconfigPath"`
	Standalone     bool   `yaml:"standalone"`
	SqlitePath     string `yaml:"sqlitePath"`
	UseFakeJob     bool   `yaml:"useFakeJob"`

	HuggingFaceAccessToken string `yaml:"huggingFaceAccessToken"`
}

type ModelStoreConfig struct {
	Enable      bool   `yaml:"enable"`
	MountPath   string `yaml:"mountPath"`
	PVClaimName string `yaml:"pvClaimName"`
}

// KubernetesManagerConfig is the Kubernetes manager configuration.
type KubernetesManagerConfig struct {
	EnableLeaderElection bool   `yaml:"enableLeaderElection"`
	LeaderElectionID     string `yaml:"leaderElectionID"`

	MetricsBindAddress string `yaml:"metricsBindAddress"`
	HealthBindAddress  string `yaml:"healthBindAddress"`
	PprofBindAddress   string `yaml:"pprofBindAddress"`
}

// Config is the configuration.
type Config struct {
	JobPollingInterval time.Duration `yaml:"jobPollingInterval"`
	JobNamespace       string        `yaml:"jobNamespace"`

	InferenceManagerAddr string `yaml:"inferenceManagerAddr"`

	ModelStore ModelStoreConfig `yaml:"modelStore"`

	Database db.Config `yaml:"database"`

	Debug DebugConfig `yaml:"debug"`

	KubernetesManager KubernetesManagerConfig `yaml:"kubernetesManager"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.JobPollingInterval <= 0 {
		return fmt.Errorf("job polling interval must be greater than 0")
	}
	if c.JobNamespace == "" {
		return fmt.Errorf("job namespace must be set")
	}
	if c.Debug.Standalone {
		if c.Debug.SqlitePath == "" {
			return fmt.Errorf("sqlite path must be set")
		}
	} else {
		if c.InferenceManagerAddr == "" {
			return fmt.Errorf("inference manager address must be set")
		}
		if err := c.Database.Validate(); err != nil {
			return fmt.Errorf("database: %s", err)
		}
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
