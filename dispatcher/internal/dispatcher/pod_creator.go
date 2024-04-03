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
	namespace string
}

func (p *PodCreator) createPod(ctx context.Context, job *store.Job) error {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("job-%s", job.JobID),
			//Labels:    job.Labels,
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
			log.Printf("Pod %s already exists\n", job.JobID)
			return nil
		}
		return err
	}
	return nil
}
