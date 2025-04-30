package dispatcher

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/dispatcher/internal/config"
	"github.com/llmariner/rbac-manager/pkg/auth"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	appsv1apply "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	metav1apply "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	notebookManagedAnnotationKey = "llmariner/managed-notebook"
	notebookIDAnnotationKey      = "llmariner/notebook-id"

	nbManagerName = "notebook-manager"

	pullingImageReason = "Pulling"
	nbContainerName    = "jupyterlab"
)

// NewNotebookManager creates a new NotebookManager
func NewNotebookManager(
	k8sClient client.Client,
	wsClient v1.WorkspaceWorkerServiceClient,
	config config.NotebooksConfig,
) *NotebookManager {
	return &NotebookManager{
		k8sClient:        k8sClient,
		wsClient:         wsClient,
		llmaBaseURL:      config.LLMarinerBaseURL,
		enablePVC:        config.EnablePVC,
		storageClassName: config.StorageClassName,
		storageSize:      config.StorageSize,
		mountPath:        config.MountPath,
	}
}

// NotebookManager is a struct that manages the notebook
type NotebookManager struct {
	k8sClient client.Client
	wsClient  v1.WorkspaceWorkerServiceClient

	llmaBaseURL string

	enablePVC        bool
	storageClassName string
	storageSize      string
	mountPath        string
}

// SetupWithManager registers the LifecycleManager with the manager.
func (n *NotebookManager) SetupWithManager(mgr ctrl.Manager) error {
	filterByAnno := (predicate.NewPredicateFuncs(func(object client.Object) bool {
		return isManagedNotebook(object.GetAnnotations())
	}))
	return ctrl.NewControllerManagedBy(mgr).
		Named("notebook").
		For(&appsv1.Deployment{}, builder.WithPredicates(filterByAnno)).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger {
			if r != nil {
				return mgr.GetLogger().WithValues("notebook", r.NamespacedName)
			}
			return mgr.GetLogger()
		}).
		Complete(n)
}

// Reconcile reconciles the notebook deployment.
func (n *NotebookManager) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var nb appsv1.Deployment
	if err := n.k8sClient.Get(ctx, req.NamespacedName, &nb); err != nil {
		log.V(2).Info("Failed to get the notebook deployment", "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !nb.DeletionTimestamp.IsZero() {
		log.V(2).Info("Notebook deployment is being deleted")
		return ctrl.Result{}, nil
	}

	replicas := ptr.Deref(nb.Spec.Replicas, 0)
	if replicas == 0 {
		log.V(4).Info("Notebook deployment is being stopped")
		return ctrl.Result{}, nil
	}

	// Get the associated Pod to check its status reason
	podList := &corev1.PodList{}
	if err := n.k8sClient.List(ctx, podList,
		client.InNamespace(req.Namespace),
		client.MatchingLabels(nb.Spec.Selector.MatchLabels)); err != nil {
		log.Error(err, "Failed to list pods for the notebook deployment")
		return ctrl.Result{}, err
	}

	reason := ""
	state := v1.NotebookState_RUNNING

	// Try to extract a more specific reason from the Pod status
	if len(podList.Items) > 0 {
		pod := podList.Items[0]
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.Name == nbContainerName {
				if containerStatus.State.Waiting != nil && strings.Contains(containerStatus.State.Waiting.Reason, pullingImageReason) {
					reason = pullingImageReason
					state = v1.NotebookState_INITIALIZING
				}
				break
			}
		}
	}

	if nb.Status.ReadyReplicas < replicas && reason == "" {
		log.V(4).Info("Notebook deployment is not ready yet")
		return ctrl.Result{}, nil
	}

	ctx = auth.AppendWorkerAuthorization(ctx)
	if _, err := n.wsClient.UpdateNotebookState(ctx, &v1.UpdateNotebookStateRequest{
		Id:     req.Name,
		State:  state,
		Reason: reason,
	}); err != nil {
		log.Error(err, "Failed to update the notebook state")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (n *NotebookManager) createNotebook(ctx context.Context, nb *v1.InternalNotebook) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Creating a deployment for a notebook")

	name := nb.Notebook.Id
	labels := map[string]string{
		"app.kubernetes.io/name":       "llma-notebook",
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/created-by": nbManagerName,
	}

	var envs []*corev1apply.EnvVarApplyConfiguration
	for k, v := range nb.Notebook.Envs {
		envs = append(envs, corev1apply.EnvVar().WithName(k).WithValue(v))
	}
	// The following sets the configuration that the OpenAI Python library uses.
	envs = append(envs,
		corev1apply.EnvVar().WithName("OPENAI_BASE_URL").WithValue(n.llmaBaseURL),
		corev1apply.EnvVar().WithName("OPENAI_ORG_ID").WithValue(nb.Notebook.OrganizationId),
		corev1apply.EnvVar().WithName("OPENAI_PROJECT_ID").WithValue(nb.Notebook.ProjectId),
	)

	req := corev1.ResourceList{}
	limit := corev1.ResourceList{}
	if r := nb.Notebook.Resources; r != nil {
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

	const (
		appPort      = 8888
		portName     = "jupyter-web-ui"
		pvcMountName = "work"
	)
	baseURL := fmt.Sprintf("/v1/sessions/%s/v1/services/notebooks/%s/%s", nb.Notebook.ClusterId, nb.Notebook.Id, nb.Notebook.KubernetesNamespace)

	containerConf := corev1apply.Container().
		WithName(nbContainerName).
		WithImage(nb.Notebook.Image).
		WithImagePullPolicy(corev1.PullIfNotPresent).
		// TODO: rethink authentication method
		WithCommand("start-notebook.py").
		WithArgs(
			"--IdentityProvider.token=$(NOTEBOOK_TOKEN)",
			"--ServerApp.base_url="+baseURL,
			// This is needed when a user accesses the notebook
			// via Session Manager/Agent and internal ingress controller.
			// TODO(kenji): Tighten this.
			"--NotebookApp.allow_origin=*").
		WithPorts(corev1apply.ContainerPort().
			WithName(portName).
			WithContainerPort(appPort).
			WithProtocol(corev1.ProtocolTCP)).
		WithEnv(envs...).
		WithEnvFrom(corev1apply.EnvFromSource().
			WithSecretRef(corev1apply.SecretEnvSource().
				WithName(nb.Notebook.Id))).
		WithResources(resources)

	podTemplateSpec := corev1apply.PodTemplateSpec().
		WithLabels(labels).
		WithSpec(corev1apply.PodSpec().
			WithContainers(containerConf))

	if n.enablePVC {
		containerConf = containerConf.
			WithVolumeMounts(corev1apply.VolumeMount().
				WithName(pvcMountName).
				WithMountPath(n.mountPath))
		podTemplateSpec = corev1apply.PodTemplateSpec().
			WithLabels(labels).
			WithSpec(corev1apply.PodSpec().
				WithContainers(containerConf).
				WithVolumes(corev1apply.Volume().
					WithName(pvcMountName).
					WithPersistentVolumeClaim(corev1apply.PersistentVolumeClaimVolumeSource().
						WithClaimName(name))))
	}

	deployConf := appsv1apply.
		Deployment(name, nb.Notebook.KubernetesNamespace).
		WithLabels(labels).
		WithAnnotations(map[string]string{
			notebookManagedAnnotationKey: "true",
			notebookIDAnnotationKey:      nb.Notebook.Id}).
		WithSpec(appsv1apply.DeploymentSpec().
			WithReplicas(1).
			WithSelector(metav1apply.LabelSelector().
				WithMatchLabels(labels)).
			WithTemplate(podTemplateSpec))

	svcConf := corev1apply.Service(name, nb.Notebook.KubernetesNamespace).
		WithLabels(labels).
		WithAnnotations(map[string]string{
			notebookManagedAnnotationKey: "true",
			notebookIDAnnotationKey:      nb.Notebook.Id}).
		WithSpec(corev1apply.ServiceSpec().
			WithType(corev1.ServiceTypeClusterIP).
			WithSelector(labels).
			WithPorts(corev1apply.ServicePort().
				WithName(portName).
				WithPort(appPort).
				WithTargetPort(intstr.FromString(portName)).
				WithProtocol(corev1.ProtocolTCP)))

	patchOpts := &client.PatchOptions{FieldManager: nbManagerName, Force: ptr.To(true)}
	deploy, err := n.applyObject(ctx, deployConf, patchOpts)
	if err != nil {
		return err
	}

	gvk := deploy.GetObjectKind().GroupVersionKind()
	ownerRef := metav1apply.OwnerReference().
		WithAPIVersion(gvk.GroupVersion().String()).
		WithKind(gvk.Kind).
		WithName(deploy.GetName()).
		WithUID(deploy.GetUID()).
		WithBlockOwnerDeletion(true).
		WithController(true)

	svcConf.WithOwnerReferences(ownerRef)

	// Secret is pre-created by server, and dispatcher only set the owner reference here.
	// TODO(aya): garbage collect orphaned secrets
	secConf := corev1apply.Secret(nb.Notebook.Id, nb.Notebook.KubernetesNamespace).
		WithOwnerReferences(ownerRef)

	objs := []any{svcConf, secConf}

	if n.enablePVC {
		pvcConf := corev1apply.PersistentVolumeClaim(name, nb.Notebook.KubernetesNamespace).
			WithLabels(labels).
			WithOwnerReferences(ownerRef).
			WithAnnotations(map[string]string{
				notebookManagedAnnotationKey: "true",
				notebookIDAnnotationKey:      nb.Notebook.Id}).
			WithSpec(corev1apply.PersistentVolumeClaimSpec().
				WithAccessModes(corev1.ReadWriteOnce).
				WithStorageClassName(n.storageClassName).
				WithResources(corev1apply.VolumeResourceRequirements().
					WithRequests(corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(n.storageSize),
					})))
		objs = append(objs, pvcConf)
	}

	for _, obj := range objs {
		if _, err := n.applyObject(ctx, obj, patchOpts); err != nil {
			return err
		}
	}
	return nil
}

func (n *NotebookManager) applyObject(ctx context.Context, applyConfig any, opts ...client.PatchOption) (client.Object, error) {
	uobj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(applyConfig)
	if err != nil {
		return nil, err
	}
	obj := &unstructured.Unstructured{Object: uobj}
	if err := n.k8sClient.Patch(ctx, obj, client.Apply, opts...); err != nil {
		return nil, err
	}
	return obj, nil
}

func (n *NotebookManager) stopNotebook(ctx context.Context, nb *v1.InternalNotebook) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Stopping a deployment for a notebook")

	var deploy appsv1.Deployment
	if err := n.k8sClient.Get(ctx, types.NamespacedName{
		Name:      nb.Notebook.Id,
		Namespace: nb.Notebook.KubernetesNamespace,
	}, &deploy); err != nil {
		return err
	}

	scale := &autoscalingv1.Scale{Spec: autoscalingv1.ScaleSpec{Replicas: 0}}
	return n.k8sClient.SubResource("scale").Update(ctx, &deploy, client.WithSubResourceBody(scale))
}

func (n *NotebookManager) deleteNotebook(ctx context.Context, nb *v1.InternalNotebook) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Deleting a deployment for a notebook")

	var deploy appsv1.Deployment
	if err := n.k8sClient.Get(ctx, types.NamespacedName{
		Name:      nb.Notebook.Id,
		Namespace: nb.Notebook.KubernetesNamespace,
	}, &deploy); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(4).Info("Deployment not found")
			return nil
		}
		return err
	}

	if err := n.k8sClient.Delete(ctx, &deploy); err == nil {
		if apierrors.IsNotFound(err) {
			log.V(4).Info("Deployment not found")
			return nil
		}
		return err
	}
	return nil
}

func isManagedNotebook(annotations map[string]string) bool {
	return annotations[notebookManagedAnnotationKey] == "true"
}
