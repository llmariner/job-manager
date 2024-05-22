package server

import (
	"context"

	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/dispatcher/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batchv1apply "k8s.io/client-go/applyconfigurations/batch/v1"
	"k8s.io/client-go/kubernetes"
)

const fieldManager = "job-manager-server"

// DefaultK8sJobClient is a client for Kubernetes Job resources.
type DefaultK8sJobClient struct {
	client kubernetes.Interface
}

// NewK8sJobClient creates a new K8sJobClient.
func NewK8sJobClient(client kubernetes.Interface) *DefaultK8sJobClient {
	return &DefaultK8sJobClient{
		client: client,
	}
}

// CancelJob cancels a job.
func (c *DefaultK8sJobClient) CancelJob(ctx context.Context, job *v1.Job, namespace string) error {
	opts := metav1.ApplyOptions{FieldManager: fieldManager, Force: true}
	name := util.GetK8sJobName(job.Id)
	conf := batchv1apply.Job(name, namespace).
		WithSpec(batchv1apply.JobSpec().
			WithSuspend(true))
	_, err := c.client.BatchV1().Jobs(namespace).Apply(ctx, conf, opts)
	return err
}
