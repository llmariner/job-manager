package config

import (
	"fmt"
	"os"
	"time"

	"github.com/llm-operator/job-manager/common/pkg/db"
	"gopkg.in/yaml.v3"
)

// S3Config is the S3 configuration.
type S3Config struct {
	EndpointURL string `yaml:"endpointUrl"`
	Bucket      string `yaml:"bucket"`
}

// ObjectStoreConfig is the object store configuration.
type ObjectStoreConfig struct {
	S3 S3Config `yaml:"s3"`
}

// Validate validates the object store configuration.
func (c *ObjectStoreConfig) Validate() error {
	if c.S3.EndpointURL == "" {
		return fmt.Errorf("s3 endpoint url must be set")
	}
	if c.S3.Bucket == "" {
		return fmt.Errorf("s3 bucket must be set")
	}
	return nil
}

// DebugConfig is the debug configuration.
type DebugConfig struct {
	KubeconfigPath string `yaml:"kubeconfigPath"`
	Standalone     bool   `yaml:"standalone"`
	SqlitePath     string `yaml:"sqlitePath"`
	UseFakeJob     bool   `yaml:"useFakeJob"`
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

	ModelManagerInternalServerAddr string `yaml:"modelManagerInternalServerAddr"`
	FileManagerInternalServerAddr  string `yaml:"fileManagerInternalServerAddr"`

	Database db.Config `yaml:"database"`

	ObjectStore ObjectStoreConfig `yaml:"objectStore"`

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
		if c.ModelManagerInternalServerAddr == "" {
			return fmt.Errorf("model manager internal server address must be set")
		}
		if c.FileManagerInternalServerAddr == "" {
			return fmt.Errorf("file manager internal server address must be set")
		}

		if err := c.ObjectStore.Validate(); err != nil {
			return fmt.Errorf("object store: %s", err)
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
