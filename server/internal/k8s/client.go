package k8s

import (
	"context"
	"fmt"

	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/rbac-manager/pkg/auth"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batchv1apply "k8s.io/client-go/applyconfigurations/batch/v1"
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
	CancelJob(ctx context.Context, job *v1.Job, namespace string) error
	CreateSecret(ctx context.Context, name, namespace string, data map[string][]byte) error
}

type defaultClient struct {
	client kubernetes.Interface
}

// CancelJob cancels a job.
func (c *defaultClient) CancelJob(ctx context.Context, job *v1.Job, namespace string) error {
	opts := metav1.ApplyOptions{FieldManager: fieldManager, Force: true}
	conf := batchv1apply.Job(job.Id, namespace).
		WithSpec(batchv1apply.JobSpec().
			WithSuspend(true))
	_, err := c.client.BatchV1().Jobs(namespace).Apply(ctx, conf, opts)
	return err
}

// CreateSecret creates a secret.
func (c *defaultClient) CreateSecret(ctx context.Context, name, namespace string, data map[string][]byte) error {
	opts := metav1.ApplyOptions{FieldManager: fieldManager, Force: true}
	conf := corev1apply.Secret(name, namespace).WithData(data)
	_, err := c.client.CoreV1().Secrets(namespace).Apply(ctx, conf, opts)
	return err
}
