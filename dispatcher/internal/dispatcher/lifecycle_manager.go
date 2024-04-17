package dispatcher

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/llm-operator/job-manager/common/pkg/store"
	"github.com/llm-operator/job-manager/dispatcher/pkg/util"
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
	Process(ctx context.Context, job *store.Job) error
}

// NoopPostProcessor is a no-op implementation of PostProcessorI.
type NoopPostProcessor struct {
}

// Process is a no-op implementation of Process.
func (p *NoopPostProcessor) Process(ctx context.Context, job *store.Job) error {
	return nil
}

// NewLifecycleManager returns a new LifecycleManager.
func NewLifecycleManager(
	store *store.S,
	client client.Client,
	postProcessor PostProcessorI,
) *LifecycleManager {
	return &LifecycleManager{
		store:         store,
		k8sClient:     client,
		postProcessor: postProcessor,
	}
}

// LifecycleManager manages job lifecycle and sync status.
type LifecycleManager struct {
	store         *store.S
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

	jobID := util.GetJobID(req.Name)
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
			if err = s.store.UpdateJobState(jobData.JobID, jobData.Version, store.JobStateQueued); err != nil {
				log.Error(err, "Failed to update job state")
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

	jobData, err := s.store.GetJobByJobID(jobID)
	if err != nil {
		log.Error(err, "Failed to get Job from jobID")
		return ctrl.Result{}, err
	}
	// TODO(aya): handle status mismatch
	switch jobData.State {
	case store.JobStateSucceeded, store.JobStatusFailed:
		// do nothing, already complete
		log.V(2).Info("Job is already completed", "state", jobData.State)
		return ctrl.Result{}, nil
	case store.JobStateCancelled:
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
		log.Info("Delete the cancelled and expired job")
		return ctrl.Result{}, s.k8sClient.Delete(ctx, &job)
	}

	if job.Status.Failed > 0 {
		if err := s.store.UpdateJobState(jobData.JobID, jobData.Version, store.JobStatusFailed); err != nil {
			log.Error(err, "Failed to update job state")
			return ctrl.Result{}, err
		}
		var message string
		for _, cond := range job.Status.Conditions {
			if cond.Type == batchv1.JobFailed {
				message = fmt.Sprintf("%s: %s", cond.Reason, cond.Message)
				break
			}
		}
		log.Info("Job failed", "msg", message)
		return ctrl.Result{}, nil
	}
	log.Info("Job successfully completed")

	log.Info("Running post-processing")
	if err := s.postProcessor.Process(ctx, jobData); err != nil {
		log.Error(err, "Failed to post process")
		return ctrl.Result{}, err
	}
	log.Info("Post-processing successfully completed")

	if err := s.store.UpdateJobState(jobData.JobID, jobData.Version, store.JobStateSucceeded); err != nil {
		log.Error(err, "Failed to update job state")
		return ctrl.Result{}, err
	}

	log.Info("Finished processing job")
	return ctrl.Result{}, nil
}
