package config

import (
	"fmt"
	"os"
	"time"

	"github.com/llm-operator/common/pkg/db"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
)

// S3Config is the S3 configuration.
type S3Config struct {
	EndpointURL string `yaml:"endpointUrl"`
	Region      string `yaml:"region"`
	Bucket      string `yaml:"bucket"`
}

// ObjectStoreConfig is the object store configuration.
type ObjectStoreConfig struct {
	S3 S3Config `yaml:"s3"`
}

// validate validates the object store configuration.
func (c *ObjectStoreConfig) validate() error {
	if c.S3.EndpointURL == "" {
		return fmt.Errorf("s3 endpoint url must be set")
	}
	if c.S3.Region == "" {
		return fmt.Errorf("s3 region must be set")
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
}

// KubernetesManagerConfig is the Kubernetes manager configuration.
type KubernetesManagerConfig struct {
	EnableLeaderElection bool   `yaml:"enableLeaderElection"`
	LeaderElectionID     string `yaml:"leaderElectionID"`

	MetricsBindAddress string `yaml:"metricsBindAddress"`
	HealthBindAddress  string `yaml:"healthBindAddress"`
	PprofBindAddress   string `yaml:"pprofBindAddress"`
}

// JobConfig is the job configuration.
type JobConfig struct {
	Image           string            `yaml:"image"`
	Version         string            `yaml:"version"`
	ImagePullPolicy corev1.PullPolicy `yaml:"imagePullPolicy"`
	NumGPUs         int               `yaml:"numGpus"`
}

// validate validates the job configuration.
func (c *JobConfig) validate() error {
	if c.Image == "" {
		return fmt.Errorf("image must be set")
	}
	if c.Version == "" {
		return fmt.Errorf("version must be set")
	}
	p := c.ImagePullPolicy
	if p == "" {
		return fmt.Errorf("image pull policy must be set")
	}
	if p != corev1.PullAlways && p != corev1.PullIfNotPresent && p != corev1.PullNever {
		return fmt.Errorf("invalid image pull policy")
	}

	if c.NumGPUs < 0 {
		return fmt.Errorf("num GPUs must be greater than or equal to 0")
	}

	return nil
}

// KueueConfig is the Kueue configuration.
type KueueConfig struct {
	Enable bool `yaml:"enable"`

	DefaultQueueName string `yaml:"defaultQueueName"`
}

// validate validates the Kueue configuration.
func (c *KueueConfig) validate() error {
	if !c.Enable {
		return nil
	}
	if c.DefaultQueueName == "" {
		return fmt.Errorf("default queue name must be set")
	}
	return nil
}

// Config is the configuration.
type Config struct {
	JobPollingInterval time.Duration `yaml:"jobPollingInterval"`

	Job JobConfig `yaml:"job"`

	ModelManagerInternalServerAddr string `yaml:"modelManagerInternalServerAddr"`
	FileManagerInternalServerAddr  string `yaml:"fileManagerInternalServerAddr"`

	Database db.Config `yaml:"database"`

	ObjectStore ObjectStoreConfig `yaml:"objectStore"`

	Debug DebugConfig `yaml:"debug"`

	KubernetesManager KubernetesManagerConfig `yaml:"kubernetesManager"`

	KueueIntegration KueueConfig `yaml:"kueueIntegration"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.JobPollingInterval <= 0 {
		return fmt.Errorf("job polling interval must be greater than 0")
	}
	if err := c.Job.validate(); err != nil {
		return fmt.Errorf("job: %s", err)
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

		if err := c.ObjectStore.validate(); err != nil {
			return fmt.Errorf("object store: %s", err)
		}

		if err := c.Database.Validate(); err != nil {
			return fmt.Errorf("database: %s", err)
		}
	}

	if err := c.KueueIntegration.validate(); err != nil {
		return fmt.Errorf("kueue integration: %s", err)
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
