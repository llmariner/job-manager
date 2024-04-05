package dispatcher

import (
	"context"
	"fmt"
	"log"

	"github.com/llm-operator/job-manager/common/pkg/store"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewPodCreator returns a new PodCreator.
func NewPodCreator(
	k8sClient kubernetes.Interface,
	namespace string,
) *PodCreator {
	return &PodCreator{
		k8sClient: k8sClient,
		namespace: namespace,
	}
}

// PodCreator creates a pod for a job.
type PodCreator struct {
	k8sClient kubernetes.Interface
	// TODO(kenji): Be able to specify the namespace per tenant.
	namespace string
}

func (p *PodCreator) createPod(ctx context.Context, job *store.Job) error {
	// TODO(kenji): Create a real fine-tuning job. See https://github.com/llm-operator/job-manager/tree/main/build/experiments/fine-tuning.
	// TODO(kenji): Be able to easily switch between real impl and fake impl. We'd like to run this code in a non-GPU node for testing.
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("job-%s", job.JobID),
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "main",
					Image: "busybox",
				},
			},
		},
	}
	if _, err := p.k8sClient.CoreV1().Pods(p.namespace).Create(ctx, pod, metav1.CreateOptions{}); err != nil {
		if apierrors.IsAlreadyExists(err) {
			// TODO(kenji): Revisit this error handling.
			log.Printf("Pod %s already exists\n", job.JobID)
			return nil
		}
		return err
	}
	return nil
}
