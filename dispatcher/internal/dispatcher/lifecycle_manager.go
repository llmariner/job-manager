package dispatcher

import (
	"context"
	"log"

	iv1 "github.com/llm-operator/inference-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ModelRegisterClient is the client for the model register service.
type ModelRegisterClient interface {
	RegisterModel(ctx context.Context, in *iv1.RegisterModelRequest, opts ...grpc.CallOption) (*iv1.RegisterModelResponse, error)
}

// NoopModelRegisterClient is a no-op implementation of ModelRegisterClient.
type NoopModelRegisterClient struct {
}

// RegisterModel is a no-op implementation of RegisterModel.
func (c *NoopModelRegisterClient) RegisterModel(ctx context.Context, in *iv1.RegisterModelRequest, opts ...grpc.CallOption) (*iv1.RegisterModelResponse, error) {
	return &iv1.RegisterModelResponse{}, nil
}

// NewLifecycleManager returns a new LifecycleManager.
func NewLifecycleManager(store *store.S, client client.Client, modelRegisterClient ModelRegisterClient) *LifecycleManager {
	return &LifecycleManager{
		store:               store,
		k8sClient:           client,
		modelRegisterClient: modelRegisterClient,
	}
}

// LifecycleManager manages job lifecycle and sync status.
type LifecycleManager struct {
	store               *store.S
	k8sClient           client.Client
	modelRegisterClient ModelRegisterClient
}

// SetupWithManager registers the LifecycleManager with the manager.
func (s *LifecycleManager) SetupWithManager(mgr ctrl.Manager) error {
	filterByAnno := (predicate.NewPredicateFuncs(func(object client.Object) bool {
		return isManagedPod(object.GetAnnotations())
	}))
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}, builder.WithPredicates(filterByAnno)).
		Complete(s)
}

// Reconcile reconciles the pod managed by llm-operator.
func (s *LifecycleManager) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	var pod corev1.Pod
	if err := s.k8sClient.Get(ctx, req.NamespacedName, &pod); err != nil {
		if apierrors.IsNotFound(err) {
			// PodGC deletes terminated pods if the number of pods exceeds the threshold (default: 12,500).
			// TODO(aya): rethink error handling & cleanup terminated pods
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if pod.Status.Phase != corev1.PodSucceeded {
		// TODO(aya): handle other pod statuses
		log.Printf("Pod %s/%s is still running.\n", pod.Namespace, pod.Name)
		return ctrl.Result{}, nil
	}

	jobID, ok := pod.GetAnnotations()[jobIDAnnotationKey]
	if !ok {
		log.Printf("Pod %s/%s is missing jobID annotation.\n", pod.Namespace, pod.Name)
		return ctrl.Result{}, nil
	}
	job, err := s.store.GetJobByJobID(jobID)
	if err != nil {
		log.Printf("Failed to get Job from jobID (jobID: %s)\n", jobID)
		return ctrl.Result{}, err
	}
	if job.State == store.JobStateCompleted {
		// do nothing, already completed
		return ctrl.Result{}, nil
	}

	// TODO(kenji): Watch pods and update job state. The code should be changed to check the completion of the job in an async fashion.
	log.Printf("Registering genereated fine-tuned model\n")
	if _, err := s.modelRegisterClient.RegisterModel(ctx, &iv1.RegisterModelRequest{
		// TODO(kenji): Fix this.
		ModelName:   "gemma:2b-fine-tuned",
		BaseModel:   "gemma:2b",
		AdapterPath: "/models/adapter/ggml-adapter-model.bin",
	}); err != nil {
		log.Printf("Failed to register model: %s\n", err)
		return ctrl.Result{}, err
	}

	if err := s.store.UpdateJobState(job.JobID, job.Version, store.JobStateCompleted); err != nil {
		log.Printf("Failed to update job state: %s\n", err)
		return ctrl.Result{}, err
	}

	log.Printf("Finished processing job (ID: %s)\n", job.JobID)
	return ctrl.Result{}, nil
}
