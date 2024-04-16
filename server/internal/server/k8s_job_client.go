package server

import (
	"context"

	"github.com/llm-operator/job-manager/dispatcher/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batchv1apply "k8s.io/client-go/applyconfigurations/batch/v1"
	"k8s.io/client-go/kubernetes"
)

const fieldManager = "job-manager-server"

// DefaultK8sJobClient is a client for Kubernetes Job resources.
type DefaultK8sJobClient struct {
	client kubernetes.Interface

	jobNamespace string
}

// NewK8sJobClient creates a new K8sJobClient.
func NewK8sJobClient(client kubernetes.Interface, jobNamespace string) *DefaultK8sJobClient {
	return &DefaultK8sJobClient{
		client:       client,
		jobNamespace: jobNamespace,
	}
}

// CancelJob cancels a job.
func (c *DefaultK8sJobClient) CancelJob(ctx context.Context, jobID string) error {
	opts := metav1.ApplyOptions{FieldManager: fieldManager}
	conf := batchv1apply.Job(util.GetK8sJobName(jobID), c.jobNamespace).
		WithSpec(batchv1apply.JobSpec().
			WithSuspend(true))
	_, err := c.client.BatchV1().Jobs(c.jobNamespace).Apply(ctx, conf, opts)
	return err
}
