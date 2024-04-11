package dispatcher

import (
	"context"

	"github.com/go-logr/logr"
	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	mv1 "github.com/llm-operator/model-manager/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ModelCreatorClient is the client for the model creation service.
type ModelCreatorClient interface {
	RegisterModel(ctx context.Context, in *mv1.RegisterModelRequest, opts ...grpc.CallOption) (*mv1.RegisterModelResponse, error)
	PublishModel(ctx context.Context, in *mv1.PublishModelRequest, opts ...grpc.CallOption) (*mv1.PublishModelResponse, error)
}

// NoopModelCreatorClient is a no-op implementation of ModelCreatorClient.
type NoopModelCreatorClient struct {
}

// RegisterModel is a no-op implementation of RegisterModel.
func (c *NoopModelCreatorClient) RegisterModel(
	ctx context.Context,
	in *mv1.RegisterModelRequest,
	opts ...grpc.CallOption,
) (*mv1.RegisterModelResponse, error) {
	return &mv1.RegisterModelResponse{}, nil
}

// PublishModel is a no-op implementation of PublishModel.
func (c *NoopModelCreatorClient) PublishModel(
	ctx context.Context,
	in *mv1.PublishModelRequest,
	opts ...grpc.CallOption,
) (*mv1.PublishModelResponse, error) {
	return &mv1.PublishModelResponse{}, nil
}

// NewLifecycleManager returns a new LifecycleManager.
func NewLifecycleManager(
	store *store.S,
	client client.Client,
	modelCreatorClient ModelCreatorClient,
) *LifecycleManager {
	return &LifecycleManager{
		store:              store,
		k8sClient:          client,
		modelCreatorClient: modelCreatorClient,
	}
}

// LifecycleManager manages job lifecycle and sync status.
type LifecycleManager struct {
	store              *store.S
	k8sClient          client.Client
	modelCreatorClient ModelCreatorClient
}

// SetupWithManager registers the LifecycleManager with the manager.
func (s *LifecycleManager) SetupWithManager(mgr ctrl.Manager) error {
	filterByAnno := (predicate.NewPredicateFuncs(func(object client.Object) bool {
		return isManagedPod(object.GetAnnotations())
	}))
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}, builder.WithPredicates(filterByAnno)).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger {
			if r != nil {
				return mgr.GetLogger().WithValues("pod", r.NamespacedName)
			}
			return mgr.GetLogger()
		}).
		Complete(s)
}

// Reconcile reconciles the pod managed by llm-operator.
func (s *LifecycleManager) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var pod corev1.Pod
	if err := s.k8sClient.Get(ctx, req.NamespacedName, &pod); err != nil {
		if apierrors.IsNotFound(err) {
			// PodGC deletes terminated pods if the number of pods exceeds the threshold (default: 12,500).
			// TODO(aya): rethink error handling & cleanup terminated pods
			log.Info("Pod not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get pod")
		return ctrl.Result{}, err
	}

	if pod.Status.Phase != corev1.PodSucceeded {
		// TODO(aya): handle other pod statuses
		log.V(2).Info("Pod is still running")
		return ctrl.Result{}, nil
	}

	jobID, ok := pod.GetAnnotations()[jobIDAnnotationKey]
	if !ok {
		log.Error(nil, "Pod is missing jobID annotation")
		return ctrl.Result{}, nil
	}
	log = log.WithValues("jobID", jobID)
	ctx = ctrl.LoggerInto(ctx, log)

	job, err := s.store.GetJobByJobID(jobID)
	if err != nil {
		log.Error(err, "Failed to get Job from jobID")
		return ctrl.Result{}, err
	}
	if job.State == store.JobStateCompleted {
		// do nothing, already completed
		return ctrl.Result{}, nil
	}
	log.Info("Pod successfully completed")

	log.Info("Registering genereated fine-tuned model")
	// TODO(kenji): Currently the model is generated at /models/adapter/ggml-adapter-model.bin. We
	// neeed to put this to a location where inference engine can retrieve.

	var jobProto v1.Job
	if err := proto.Unmarshal(job.Message, &jobProto); err != nil {
		log.Error(err, "Failed to unmarshal job")
		return ctrl.Result{}, err
	}
	resp, err := s.modelCreatorClient.RegisterModel(ctx, &mv1.RegisterModelRequest{
		BaseModel: jobProto.Model,
		Suffix:    job.Suffix,
		TenantId:  job.TenantID,
	})
	if err != nil {
		log.Error(err, "Failed to register model")
		return ctrl.Result{}, err
	}

	// TODO(kenji): Upload the model to the specified location.

	if _, err := s.modelCreatorClient.PublishModel(ctx, &mv1.PublishModelRequest{
		Id:       resp.Id,
		TenantId: job.TenantID,
	}); err != nil {
		log.Error(err, "Failed to publish model")
		return ctrl.Result{}, err
	}

	// TODO: Update the status and the fined-tuned model in the Message.

	if err := s.store.UpdateJobState(job.JobID, job.Version, store.JobStateCompleted); err != nil {
		log.Error(err, "Failed to update job state")
		return ctrl.Result{}, err
	}

	log.Info("Finished processing job")
	return ctrl.Result{}, nil
}
