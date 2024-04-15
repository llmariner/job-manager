package dispatcher

import (
	"context"
	"time"

	"github.com/llm-operator/job-manager/common/pkg/store"
	ctrl "sigs.k8s.io/controller-runtime"
)

type jobCreatorI interface {
	createJob(ctx context.Context, job *store.Job) error
}

// New returns a new dispatcher.
func New(
	store *store.S,
	podCreator jobCreatorI,
	jobPollingInterval time.Duration,
) *D {
	return &D{
		store:              store,
		jobCreator:         podCreator,
		jobPollingInterval: jobPollingInterval,
	}
}

// D is a dispatcher.
type D struct {
	store      *store.S
	jobCreator jobCreatorI

	jobPollingInterval time.Duration
}

// SetupWithManager registers the dispatcher with the manager.
func (d *D) SetupWithManager(mgr ctrl.Manager) error {
	return mgr.Add(d)
}

// Start starts the dispatcher.
func (d *D) Start(ctx context.Context) error {
	if err := d.processPendingJobs(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(d.jobPollingInterval)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := d.processPendingJobs(ctx); err != nil {
				return err
			}
		}
	}
}

func (d *D) processPendingJobs(ctx context.Context) error {
	jobs, err := d.store.ListPendingJobs()
	if err != nil {
		return err
	}

	for _, job := range jobs {
		if err := d.processJob(ctx, job); err != nil {
			return err
		}
	}
	return nil
}

func (d *D) processJob(ctx context.Context, job *store.Job) error {
	log := ctrl.LoggerFrom(ctx).WithValues("jobID", job.JobID)
	ctx = ctrl.LoggerInto(ctx, log)

	log.Info("Processing job")
	if err := d.jobCreator.createJob(ctx, job); err != nil {
		return err
	}
	return d.store.UpdateJobState(job.JobID, job.Version, store.JobStateRunning)
}
