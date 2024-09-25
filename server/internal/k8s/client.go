package k8s

import (
	"context"
	"fmt"

	"github.com/llmariner/rbac-manager/pkg/auth"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const fieldManager = "job-manager-server"

// ClientFactory is a factory to create a Client.
type ClientFactory interface {
	NewClient(env auth.AssignedKubernetesEnv, token string) (Client, error)
}

// NewClientFactory creates a new ClientFactory.
func NewClientFactory(endpoint string) ClientFactory {
	return &defaultClientFactory{endpoint: endpoint}
}

type defaultClientFactory struct {
	endpoint string
}

// NewK8sClient creates a new Client.
func (f *defaultClientFactory) NewClient(env auth.AssignedKubernetesEnv, token string) (Client, error) {
	client, err := kubernetes.NewForConfig(&rest.Config{
		Host:        fmt.Sprintf("%s/sessions/%s", f.endpoint, env.ClusterID),
		BearerToken: token,
	})
	if err != nil {
		return nil, err
	}
	return &defaultClient{client: client}, nil
}

// Client is a client to mange worker Kubernetes resources.
type Client interface {
	CreateSecret(ctx context.Context, name, namespace string, data map[string][]byte) error
	CreateConfigMap(ctx context.Context, name, namespace string, data map[string][]byte) error
}

type defaultClient struct {
	client kubernetes.Interface
}

// CreateSecret creates a secret.
func (c *defaultClient) CreateSecret(ctx context.Context, name, namespace string, data map[string][]byte) error {
	opts := metav1.ApplyOptions{FieldManager: fieldManager, Force: true}
	conf := corev1apply.Secret(name, namespace).WithData(data)
	_, err := c.client.CoreV1().Secrets(namespace).Apply(ctx, conf, opts)
	return err
}

// CreateConfigMap creates a configmap.
func (c *defaultClient) CreateConfigMap(ctx context.Context, name, namespace string, data map[string][]byte) error {
	opts := metav1.ApplyOptions{FieldManager: fieldManager, Force: true}
	conf := corev1apply.ConfigMap(name, namespace).WithBinaryData(data)
	_, err := c.client.CoreV1().ConfigMaps(namespace).Apply(ctx, conf, opts)
	return err
}
