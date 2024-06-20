package config

import (
	"fmt"
	"os"
	"time"

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
}

// KubernetesManagerConfig is the Kubernetes manager configuration.
type KubernetesManagerConfig struct {
	EnableLeaderElection bool   `yaml:"enableLeaderElection"`
	LeaderElectionID     string `yaml:"leaderElectionID"`

	MetricsBindAddress string `yaml:"metricsBindAddress"`
	HealthBindAddress  string `yaml:"healthBindAddress"`
	PprofBindAddress   string `yaml:"pprofBindAddress"`
}

// WandbAPIKeySecretConfig is the W&B API key secret configuration.
type WandbAPIKeySecretConfig struct {
	Name string `yaml:"name"`
	Key  string `yaml:"key"`
}

// JobConfig is the job configuration.
type JobConfig struct {
	Image           string            `yaml:"image"`
	Version         string            `yaml:"version"`
	ImagePullPolicy corev1.PullPolicy `yaml:"imagePullPolicy"`
	NumGPUs         int               `yaml:"numGpus"`

	WandbAPIKeySecret WandbAPIKeySecretConfig `yaml:"wandbApiKeySecret"`
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

// NotebooksConfig is the notebooks configuration.
type NotebooksConfig struct {
	LLMOperatorBaseURL string `yaml:"llmOperatorBaseUrl"`
	IngressClassName   string `yaml:"ingressClassName"`
}

// validate validates the notebooks configuration.
func (c *NotebooksConfig) validate() error {
	if c.LLMOperatorBaseURL == "" {
		return fmt.Errorf("llm operator base url must be set")
	}
	if c.IngressClassName == "" {
		return fmt.Errorf("ingress class name must be set")
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

// WorkerTLSConfig is the worker TLS configuration.
type WorkerTLSConfig struct {
	Enable bool `yaml:"enable"`
}

// WorkerConfig is the worker configuration.
type WorkerConfig struct {
	TLS WorkerTLSConfig `yaml:"tls"`
}

// Config is the configuration.
type Config struct {
	PollingInterval time.Duration `yaml:"pollingInterval"`

	Job      JobConfig       `yaml:"job"`
	Notebook NotebooksConfig `yaml:"notebook"`

	ClusterManagerServerWorkerServiceAddr string `yaml:"clusterManagerServerWorkerServiceAddr"`
	JobManagerServerWorkerServiceAddr     string `yaml:"jobManagerServerWorkerServiceAddr"`
	FileManagerServerWorkerServiceAddr    string `yaml:"fileManagerServerWorkerServiceAddr"`
	ModelManagerServerWorkerServiceAddr   string `yaml:"modelManagerServerWorkerServiceAddr"`

	ObjectStore ObjectStoreConfig `yaml:"objectStore"`

	Debug DebugConfig `yaml:"debug"`

	KubernetesManager KubernetesManagerConfig `yaml:"kubernetesManager"`

	KueueIntegration KueueConfig `yaml:"kueueIntegration"`

	Worker WorkerConfig `yaml:"workerConfig"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.PollingInterval <= 0 {
		return fmt.Errorf("job polling interval must be greater than 0")
	}
	if err := c.Job.validate(); err != nil {
		return fmt.Errorf("job: %s", err)
	}
	if err := c.Notebook.validate(); err != nil {
		return fmt.Errorf("notebook: %s", err)
	}

	if !c.Debug.Standalone {
		if c.ClusterManagerServerWorkerServiceAddr == "" {
			return fmt.Errorf("cluster manager server worker service address must be set")
		}
		if c.JobManagerServerWorkerServiceAddr == "" {
			return fmt.Errorf("job manager server worker service address must be set")
		}
		if c.FileManagerServerWorkerServiceAddr == "" {
			return fmt.Errorf("file manager server worker service address must be set")
		}
		if c.ModelManagerServerWorkerServiceAddr == "" {
			return fmt.Errorf("model manager server worker service address must be set")
		}

		if err := c.ObjectStore.validate(); err != nil {
			return fmt.Errorf("object store: %s", err)
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
