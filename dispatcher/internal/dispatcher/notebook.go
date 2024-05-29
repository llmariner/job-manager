package dispatcher

import (
	"context"

	"github.com/llm-operator/job-manager/common/pkg/store"
	"github.com/llm-operator/job-manager/dispatcher/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	appsv1apply "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	metav1apply "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	managedAnnotationKey    = "llm-operator/managed"
	notebookIDAnnotationKey = "llm-operator/notebook-id"

	nbManagerName = "notebook-manager"
)

// NewNotebookManager creates a new NotebookManager
func NewNotebookManager(
	k8sClient client.Client,
	llmoBaseURL string,
) *NotebookManager {
	return &NotebookManager{
		k8sClient:   k8sClient,
		llmoBaseURL: llmoBaseURL,
	}
}

// NotebookManager is a struct that manages the notebook
type NotebookManager struct {
	k8sClient client.Client

	llmoBaseURL string
}

func (n *NotebookManager) createNotebook(ctx context.Context, nb *store.Notebook) error {
	log := ctrl.LoggerFrom(ctx)

	// TOOD: create additional resources for the notebook

	log.Info("Creating a deployment for a notebook")

	nbProto, err := nb.V1Notebook()
	if err != nil {
		return err
	}

	name := util.GetK8sNotebookName(nb.NotebookID)
	labels := map[string]string{
		"app.kubernetes.io/name":       "llmo-notebook",
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/created-by": nbManagerName,
	}

	var envs []*corev1apply.EnvVarApplyConfiguration
	for k, v := range nbProto.Envs {
		envs = append(envs, corev1apply.EnvVar().WithName(k).WithValue(v))
	}
	envs = append(envs, corev1apply.EnvVar().WithName("OPENAI_BASE_URL").WithValue(n.llmoBaseURL))
	// TODO: think the safe way to pass the user api key as the `OPENAI_API_KEY` envar.

	req := corev1.ResourceList{}
	limit := corev1.ResourceList{}
	if r := nbProto.Resources; r != nil {
		if cpu := r.CpuMilicore; cpu != nil {
			if cpu.Requests > 0 {
				req[corev1.ResourceCPU] = *resource.NewMilliQuantity(int64(r.CpuMilicore.Requests), resource.DecimalSI)
			}
			if cpu.Limits > 0 {
				limit[corev1.ResourceCPU] = *resource.NewMilliQuantity(int64(r.CpuMilicore.Limits), resource.DecimalSI)
			}
		}
		if mem := r.MemoryMegabytes; mem != nil {
			if mem.Requests > 0 {
				req[corev1.ResourceMemory] = *resource.NewScaledQuantity(int64(mem.Requests), 6)
			}
			if mem.Limits > 0 {
				limit[corev1.ResourceMemory] = *resource.NewScaledQuantity(int64(mem.Limits), 6)
			}
		}
		if r.GpuCount > 0 {
			limit["nvidia.com/gpu"] = *resource.NewQuantity(int64(r.GpuCount), resource.DecimalSI)
		}
	}
	resources := corev1apply.ResourceRequirements()
	if len(req) > 0 {
		resources.WithRequests(req)
	}
	if len(limit) > 0 {
		resources.WithLimits(limit)
	}

	// TODO: set volume mounts and volumes for the notebook

	obj := appsv1apply.
		Deployment(name, nb.KubernetesNamespace).
		WithLabels(labels).
		WithAnnotations(map[string]string{
			managedAnnotationKey:    "true",
			notebookIDAnnotationKey: nb.NotebookID}).
		WithSpec(appsv1apply.DeploymentSpec().
			WithSelector(metav1apply.LabelSelector().
				WithMatchLabels(labels)).
			WithTemplate(corev1apply.PodTemplateSpec().
				WithLabels(labels).
				WithSpec(corev1apply.PodSpec().
					WithContainers(corev1apply.Container().
						WithName("jupyterlab").
						WithImage(nb.Image).
						WithImagePullPolicy(corev1.PullIfNotPresent).
						// TODO: rethink authentication method
						WithCommand("start-notebook.py").
						WithArgs("--IdentityProvider.token=''").
						WithPorts(corev1apply.ContainerPort().
							WithName("web").
							WithContainerPort(8888).
							WithProtocol(corev1.ProtocolTCP)).
						WithEnv(envs...).
						WithResources(resources)))))

	uobj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return err
	}
	patch := &unstructured.Unstructured{Object: uobj}
	opts := &client.PatchOptions{FieldManager: nbManagerName, Force: ptr.To(true)}
	return n.k8sClient.Patch(ctx, patch, client.Apply, opts)
}
