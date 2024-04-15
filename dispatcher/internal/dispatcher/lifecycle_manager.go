package dispatcher

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/go-logr/logr"
	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	mv1 "github.com/llm-operator/model-manager/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	batchv1 "k8s.io/api/batch/v1"
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

// S3Client is an interface for an S3 client.
type S3Client interface {
	Upload(r io.Reader, key string) error
}

// NoopS3Client is a no-op S3 client.
type NoopS3Client struct{}

// Upload is a no-op implementation of Upload.
func (n *NoopS3Client) Upload(r io.Reader, key string) error {
	return nil
}

// NewLifecycleManager returns a new LifecycleManager.
func NewLifecycleManager(
	store *store.S,
	client client.Client,
	modelCreatorClient ModelCreatorClient,
	s3Client S3Client,
) *LifecycleManager {
	return &LifecycleManager{
		store:              store,
		k8sClient:          client,
		modelCreatorClient: modelCreatorClient,
		s3Client:           s3Client,
	}
}

// LifecycleManager manages job lifecycle and sync status.
type LifecycleManager struct {
	store              *store.S
	k8sClient          client.Client
	modelCreatorClient ModelCreatorClient
	s3Client           S3Client
}

// SetupWithManager registers the LifecycleManager with the manager.
func (s *LifecycleManager) SetupWithManager(mgr ctrl.Manager) error {
	filterByAnno := (predicate.NewPredicateFuncs(func(object client.Object) bool {
		return isManagedJob(object.GetAnnotations())
	}))
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.Job{}, builder.WithPredicates(filterByAnno)).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger {
			if r != nil {
				return mgr.GetLogger().WithValues("job", r.NamespacedName)
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

	jobID := strings.TrimPrefix(req.Name, jobPrefix)
	log = log.WithValues("jobID", jobID)
	ctx = ctrl.LoggerInto(ctx, log)

	var job batchv1.Job
	if err := s.k8sClient.Get(ctx, req.NamespacedName, &job); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "Failed to get pod")
			return ctrl.Result{}, err
		}

		jobData, err := s.store.GetJobByJobID(jobID)
		if err != nil {
			log.Error(err, "Failed to get Job from jobID")
			return ctrl.Result{}, err
		}
		if jobData.State == store.JobStateRunning {
			// set back to the pending status if job is accidentally deleted.
			if err = s.store.UpdateJobState(jobData.JobID, jobData.Version, store.JobStatePending); err != nil {
				log.Error(err, "Failed to update job state")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if job.Status.Succeeded == 0 {
		// TODO(aya): handle other job statuses
		log.V(2).Info("Job is still running")
		return ctrl.Result{}, nil
	}

	jobData, err := s.store.GetJobByJobID(jobID)
	if err != nil {
		log.Error(err, "Failed to get Job from jobID")
		return ctrl.Result{}, err
	}
	if jobData.State == store.JobStateCompleted {
		// do nothing, already completed
		return ctrl.Result{}, nil
	}
	log.Info("Job successfully completed")

	log.Info("Registering genereated fine-tuned model")
	var jobProto v1.Job
	if err := proto.Unmarshal(jobData.Message, &jobProto); err != nil {
		log.Error(err, "Failed to unmarshal job")
		return ctrl.Result{}, err
	}
	resp, err := s.modelCreatorClient.RegisterModel(ctx, &mv1.RegisterModelRequest{
		BaseModel: jobProto.Model,
		Suffix:    jobData.Suffix,
		TenantId:  jobData.TenantID,
	})
	if err != nil {
		log.Error(err, "Failed to register model")
		return ctrl.Result{}, err
	}

	log.Info("Uploading the model.")
	// TODO(kenji): Provide a unique location per model. Or make the job just upload the model directly.
	r, err := os.Open("/models/adapter/ggml-adapter-model.bin")
	if err != nil {
		log.Error(err, "Failed to open model")
		return ctrl.Result{}, err
	}
	if err := s.s3Client.Upload(r, resp.Path); err != nil {
		log.Error(err, "Failed to upload model")
		return ctrl.Result{}, err
	}
	log.Info("Uploaded the model successfully")

	if _, err := s.modelCreatorClient.PublishModel(ctx, &mv1.PublishModelRequest{
		Id:       resp.Id,
		TenantId: jobData.TenantID,
	}); err != nil {
		log.Error(err, "Failed to publish model")
		return ctrl.Result{}, err
	}

	// TODO: Update the status and the fined-tuned model in the Message.

	if err := s.store.UpdateJobState(jobData.JobID, jobData.Version, store.JobStateCompleted); err != nil {
		log.Error(err, "Failed to update job state")
		return ctrl.Result{}, err
	}

	log.Info("Finished processing job")
	return ctrl.Result{}, nil
}
