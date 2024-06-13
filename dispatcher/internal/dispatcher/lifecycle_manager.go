package dispatcher

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/rbac-manager/pkg/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// PostProcessorI is an interface for post-processing.
type PostProcessorI interface {
	Process(ctx context.Context, job *v1.InternalJob) error
}

// NoopPostProcessor is a no-op implementation of PostProcessorI.
type NoopPostProcessor struct {
}

// Process is a no-op implementation of Process.
func (p *NoopPostProcessor) Process(ctx context.Context, job *v1.InternalJob) error {
	return nil
}

// NewLifecycleManager returns a new LifecycleManager.
func NewLifecycleManager(
	ftClient v1.FineTuningWorkerServiceClient,
	client client.Client,
	postProcessor PostProcessorI,
) *LifecycleManager {
	return &LifecycleManager{
		ftClient:      ftClient,
		k8sClient:     client,
		postProcessor: postProcessor,
	}
}

// LifecycleManager manages job lifecycle and sync status.
type LifecycleManager struct {
	ftClient      v1.FineTuningWorkerServiceClient
	k8sClient     client.Client
	postProcessor PostProcessorI
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

	jobID := req.Name
	log = log.WithValues("jobID", jobID)
	ctx = ctrl.LoggerInto(ctx, log)
	ctx = auth.AppendWorkerAuthorization(ctx)

	var job batchv1.Job
	if err := s.k8sClient.Get(ctx, req.NamespacedName, &job); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "Failed to get pod")
			return ctrl.Result{}, err
		}

		ijob, err := s.ftClient.GetInternalJob(ctx, &v1.GetInternalJobRequest{Id: jobID})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				log.Info("Job not found in both store and k8s")
				return ctrl.Result{}, nil
			}
			log.Error(err, "Failed to get Job from jobID")
			return ctrl.Result{}, err
		}
		if ijob.State == v1.InternalJob_RUNNING {
			// set back to the pending status if job is accidentally deleted.
			if _, err = s.ftClient.UpdateJobPhase(ctx, &v1.UpdateJobPhaseRequest{
				Id:    jobID,
				Phase: v1.UpdateJobPhaseRequest_REQUEUE,
			}); err != nil {
				log.Error(err, "Failed to update job phase")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if job.Status.Succeeded == 0 && job.Status.Failed == 0 {
		// TODO(aya): handle other job statuses
		log.V(2).Info("Job is still running")
		return ctrl.Result{}, nil
	}

	ijob, err := s.ftClient.GetInternalJob(ctx, &v1.GetInternalJobRequest{Id: jobID})
	if err != nil {
		log.Error(err, "Failed to get Job from jobID")
		return ctrl.Result{}, err
	}
	switch ijob.State {
	case v1.InternalJob_RUNNING:
		// valid state, continue
	case v1.InternalJob_SUCCEEDED, v1.InternalJob_FAILED:
		// do nothing, already complete
		log.V(2).Info("Job is already completed", "state", ijob.State)
		return ctrl.Result{}, nil
	case v1.InternalJob_CANCELED:
		// TODO(aya): rethink cleanup method (e.g., post-processed data)
		var (
			expired        bool
			expirationTime time.Time
		)
		for _, cond := range job.Status.Conditions {
			if cond.Type == batchv1.JobSuspended {
				expirationTime = cond.LastTransitionTime.Add(jobTTL)
				expired = time.Now().After(expirationTime)
			}
		}
		if !expired {
			requeueAfter := time.Until(expirationTime)
			log.V(2).Info("Job is cancelled but not expired yet", "requeue-after", requeueAfter)
			return ctrl.Result{Requeue: true, RequeueAfter: requeueAfter}, nil
		}
		log.Info("Deleting the cancelled and expired job")
		return ctrl.Result{}, s.k8sClient.Delete(ctx, &job)
	default:
		// queued, unspecifed, or unknown are not valid states
		// this error could not be recovered by k8s reconciliation, so just log and return
		log.Error(fmt.Errorf("unexpected job state: %v", ijob.State), "Job state is invalid")
		return ctrl.Result{}, nil
	}

	if job.Status.Failed > 0 {
		var message string
		for _, cond := range job.Status.Conditions {
			if cond.Type == batchv1.JobFailed {
				message = fmt.Sprintf("%s: %s", cond.Reason, cond.Message)
				break
			}
		}
		if _, err = s.ftClient.UpdateJobPhase(ctx, &v1.UpdateJobPhaseRequest{
			Id:      jobID,
			Phase:   v1.UpdateJobPhaseRequest_FAILED,
			Message: message,
		}); err != nil {
			log.Error(err, "Failed to update job phase")
			return ctrl.Result{}, err
		}
		log.Info("Job failed", "msg", message)
		return ctrl.Result{}, nil
	}
	log.Info("Job successfully completed")

	log.Info("Running post-processing")
	if err := s.postProcessor.Process(ctx, ijob); err != nil {
		log.Error(err, "Failed to post process")
		return ctrl.Result{}, err
	}
	log.Info("Post-processing successfully completed")

	if _, err = s.ftClient.UpdateJobPhase(ctx, &v1.UpdateJobPhaseRequest{
		Id:      jobID,
		Phase:   v1.UpdateJobPhaseRequest_FINETUNED,
		ModelId: ijob.OutputModelId,
	}); err != nil {
		log.Error(err, "Failed to update job phase")
		return ctrl.Result{}, err
	}
	log.Info("Finished processing job")
	return ctrl.Result{}, nil
}
