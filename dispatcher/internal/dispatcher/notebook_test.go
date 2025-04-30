package dispatcher

import (
	"context"
	"testing"

	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/dispatcher/internal/config"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileNotebook(t *testing.T) {
	var tests = []struct {
		name string

		state            v1.NotebookState
		mutateNotebookFn func(nb *appsv1.Deployment)
		mutatePodFn      func(pod *corev1.Pod)

		wantUpdate bool
		wantState  v1.NotebookState
	}{
		{
			name:  "notebook is ready",
			state: v1.NotebookState_INITIALIZING,
			mutateNotebookFn: func(nb *appsv1.Deployment) {
				nb.Spec.Replicas = ptr.To(int32(1))
				nb.Status.ReadyReplicas = 1
				// Add labels that will be used by the Pod selector
				if nb.Spec.Selector == nil {
					nb.Spec.Selector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/instance": nb.Name,
						},
					}
				}
			},
			mutatePodFn: func(pod *corev1.Pod) {
				// Set Running status
				pod.Status.Phase = corev1.PodRunning
				// Add container status
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{
					{
						Name: "jupyterlab",
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
				}
			},
			wantUpdate: true,
			wantState:  v1.NotebookState_RUNNING,
		},
		{
			name:  "notebook is pulling image",
			state: v1.NotebookState_INITIALIZING,
			mutateNotebookFn: func(nb *appsv1.Deployment) {
				nb.Spec.Replicas = ptr.To(int32(1))
				nb.Status.ReadyReplicas = 0
				// Add labels that will be used by the Pod selector
				if nb.Spec.Selector == nil {
					nb.Spec.Selector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/instance": nb.Name,
						},
					}
				}
			},
			mutatePodFn: func(pod *corev1.Pod) {
				// Set Pending status
				pod.Status.Phase = corev1.PodPending
				// Add container status with waiting state and PullingImage reason
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{
					{
						Name: "jupyterlab",
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{
								Reason: "PullingImage",
							},
						},
					},
				}
			},
			wantUpdate: true,
			wantState:  v1.NotebookState_INITIALIZING,
		},
		{
			name:  "notebook is not ready",
			state: v1.NotebookState_INITIALIZING,
			mutateNotebookFn: func(nb *appsv1.Deployment) {
				nb.Spec.Replicas = ptr.To(int32(1))
				nb.Status.ReadyReplicas = 0
				// Add labels that will be used by the Pod selector
				if nb.Spec.Selector == nil {
					nb.Spec.Selector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/instance": nb.Name,
						},
					}
				}
			},
		},
		{
			name:  "stopping",
			state: v1.NotebookState_STOPPED,
			mutateNotebookFn: func(nb *appsv1.Deployment) {
				nb.Spec.Replicas = ptr.To(int32(0))
				// Add labels that will be used by the Pod selector
				if nb.Spec.Selector == nil {
					nb.Spec.Selector = &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/instance": nb.Name,
						},
					}
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nb := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nb",
					Namespace: "default",
				},
			}
			if test.mutateNotebookFn != nil {
				test.mutateNotebookFn(nb)
			}

			// Create a Pod for the Deployment
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nb-pod",
					Namespace: "default",
					Labels: map[string]string{
						"app.kubernetes.io/instance": "test-nb",
					},
				},
			}

			if test.mutatePodFn != nil {
				test.mutatePodFn(pod)
			}

			// Add both the deployment and pod to the fake client
			k8sClient := fake.NewFakeClient(nb, pod)

			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      nb.Name,
					Namespace: nb.Namespace,
				},
			}

			inbs := []*v1.InternalNotebook{
				{
					Notebook: &v1.Notebook{Id: nb.Name},
					State:    test.state,
				},
			}
			wsClient := &fakeWorkspaceWorkerServiceClient{
				notebooks:    inbs,
				updatedState: map[string]v1.NotebookState{},
			}

			mgr := NewNotebookManager(k8sClient, wsClient, config.NotebooksConfig{})
			_, err := mgr.Reconcile(context.Background(), req)
			assert.NoError(t, err)

			gotState, ok := wsClient.updatedState[nb.Name]
			assert.Equal(t, test.wantUpdate, ok)
			assert.Equal(t, test.wantState, gotState)
		})
	}
}
