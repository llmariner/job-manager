package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/llmariner/cluster-manager/pkg/status"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	kyaml "sigs.k8s.io/yaml"
)

// AssumeRoleConfig is the assume role configuration.
type AssumeRoleConfig struct {
	RoleARN    string `yaml:"roleArn"`
	ExternalID string `yaml:"externalId"`
}

func (c *AssumeRoleConfig) validate() error {
	if c.RoleARN == "" {
		return fmt.Errorf("roleArn must be set")
	}
	return nil
}

// S3Config is the S3 configuration.
type S3Config struct {
	EndpointURL        string `yaml:"endpointUrl"`
	Region             string `yaml:"region"`
	InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
	Bucket             string `yaml:"bucket"`

	AssumeRole *AssumeRoleConfig `yaml:"assumeRole"`
}

// ObjectStoreConfig is the object store configuration.
type ObjectStoreConfig struct {
	S3 S3Config `yaml:"s3"`
}

// validate validates the object store configuration.
func (c *ObjectStoreConfig) validate() error {
	if c.S3.Region == "" {
		return fmt.Errorf("s3 region must be set")
	}
	if c.S3.Bucket == "" {
		return fmt.Errorf("s3 bucket must be set")
	}
	if ar := c.S3.AssumeRole; ar != nil {
		if err := ar.validate(); err != nil {
			return fmt.Errorf("assumeRole: %s", err)
		}
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

func (c *KubernetesManagerConfig) validate() error {
	if c.EnableLeaderElection && c.LeaderElectionID == "" {
		return fmt.Errorf("leader election ID must be set")
	}
	return nil
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

	WandbAPIKeySecret WandbAPIKeySecretConfig `yaml:"wandbApiKeySecret"`

	// UseBitsAndBytesQuantization is a flag to enable bits and bytes quantization.
	UseBitsAndBytesQuantization bool `yaml:"useBitsAndBytesQuantization"`
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
	return nil
}

// NotebooksConfig is the notebooks configuration.
type NotebooksConfig struct {
	LLMarinerBaseURL string `yaml:"llmarinerBaseUrl"`
	EnablePVC        bool   `yaml:"enablePvc"`
	StorageClassName string `yaml:"storageClassName"`
	StorageSize      string `yaml:"storageSize"`
	MountPath        string `yaml:"mountPath"`
	GrantSudo        bool   `yaml:"grantSudo"`
}

// validate validates the notebooks configuration.
func (c *NotebooksConfig) validate() error {
	if c.LLMarinerBaseURL == "" {
		return fmt.Errorf("llmariner base url must be set")
	}
	if c.EnablePVC {
		if c.StorageClassName == "" {
			return fmt.Errorf("storage class name must be set")
		}
		if c.StorageSize == "" {
			return fmt.Errorf("storage size must be set")
		}
		if _, err := resource.ParseQuantity(c.StorageSize); err != nil {
			return fmt.Errorf("invalid storage size: %s", err)
		}

		if c.MountPath == "" {
			return fmt.Errorf("mount path must be set")
		}
		if !strings.HasPrefix(c.MountPath, "/") {
			return fmt.Errorf("mount path must start with a slash")
		}
	}
	return nil
}

// TolerationConfig is the toleration configuration.
type TolerationConfig struct {
	Key               string `yaml:"key"`
	Operator          string `yaml:"operator"`
	Value             string `yaml:"value"`
	Effect            string `yaml:"effect"`
	TolerationSeconds int64  `yaml:"tolerationSeconds"`
}

// WorkloadConfig is the workload configuration.
type WorkloadConfig struct {
	PodAnnotations map[string]string `yaml:"podAnnotations"`

	NodeSelector         map[string]string  `yaml:"nodeSelector"`
	Tolerations          []TolerationConfig `yaml:"tolerations"`
	UnstructuredAffinity any                `yaml:"affinity"`
	Affinity             *corev1.Affinity   `yaml:"-"`
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

	Workload WorkloadConfig `yaml:"workloadConfig"`

	JobManagerServerWorkerServiceAddr   string `yaml:"jobManagerServerWorkerServiceAddr"`
	FileManagerServerWorkerServiceAddr  string `yaml:"fileManagerServerWorkerServiceAddr"`
	ModelManagerServerWorkerServiceAddr string `yaml:"modelManagerServerWorkerServiceAddr"`

	ObjectStore ObjectStoreConfig `yaml:"objectStore"`

	Debug DebugConfig `yaml:"debug"`

	KubernetesManager KubernetesManagerConfig `yaml:"kubernetesManager"`

	KueueIntegration KueueConfig `yaml:"kueueIntegration"`

	Worker WorkerConfig `yaml:"worker"`

	// ComponentStatusSender is the configuration for the component status sender.
	ComponentStatusSender status.Config `yaml:"componentStatusSender"`

	ClusterStatusUpdateInterval time.Duration `yaml:"clusterStatusUpdateInterval"`
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

	if err := c.KubernetesManager.validate(); err != nil {
		return fmt.Errorf("kubernetes manager: %s", err)
	}

	if err := c.KueueIntegration.validate(); err != nil {
		return fmt.Errorf("kueue integration: %s", err)
	}

	if err := c.ComponentStatusSender.Validate(); err != nil {
		return fmt.Errorf("componentStatusSender: %s", err)
	}

	if c.ClusterStatusUpdateInterval <= 0 {
		return fmt.Errorf("cluster status update interval must be greater than 0")
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

	if a := config.Workload.UnstructuredAffinity; a != nil {
		data, err := yaml.Marshal(a)
		if err != nil {
			return config, fmt.Errorf("config: marshal affinity: %s", err)
		}
		var affinity corev1.Affinity
		if err := kyaml.Unmarshal(data, &affinity); err != nil {
			return config, fmt.Errorf("config: unmarshal affinity: %s", err)
		}
		config.Workload.Affinity = &affinity
	}

	return config, nil
}
