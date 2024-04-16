package dispatcher

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	"github.com/llm-operator/job-manager/common/pkg/store"
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

	log.Info("Running post-processing")
	if err := s.postProcessor.Process(ctx, jobData); err != nil {
		log.Error(err, "Failed to post process")
		return ctrl.Result{}, err
	}
	log.Info("Post-processing successfully completed")

	if err := s.store.UpdateJobState(jobData.JobID, jobData.Version, store.JobStateCompleted); err != nil {
		log.Error(err, "Failed to update job state")
		return ctrl.Result{}, err
	}

	log.Info("Finished processing job")
	return ctrl.Result{}, nil
}
