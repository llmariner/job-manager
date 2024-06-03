package server

import (
	"context"

	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/dispatcher/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batchv1apply "k8s.io/client-go/applyconfigurations/batch/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/kubernetes"
)

const fieldManager = "job-manager-server"

// DefaultK8sClient is a client for Kubernetes Job resources.
type DefaultK8sClient struct {
	client kubernetes.Interface
}

// NewK8sClient creates a new K8sJobClient.
func NewK8sClient(client kubernetes.Interface) *DefaultK8sClient {
	return &DefaultK8sClient{
		client: client,
	}
}

// CancelJob cancels a job.
func (c *DefaultK8sClient) CancelJob(ctx context.Context, job *v1.Job, namespace string) error {
	opts := metav1.ApplyOptions{FieldManager: fieldManager, Force: true}
	name := util.GetK8sJobName(job.Id)
	conf := batchv1apply.Job(name, namespace).
		WithSpec(batchv1apply.JobSpec().
			WithSuspend(true))
	_, err := c.client.BatchV1().Jobs(namespace).Apply(ctx, conf, opts)
	return err
}

// CreateSecret creates a secret.
func (c *DefaultK8sClient) CreateSecret(ctx context.Context, name, namespace string, data map[string][]byte) error {
	opts := metav1.ApplyOptions{FieldManager: fieldManager, Force: true}
	conf := corev1apply.Secret(name, namespace).WithData(data)
	_, err := c.client.CoreV1().Secrets(namespace).Apply(ctx, conf, opts)
	return err
}
