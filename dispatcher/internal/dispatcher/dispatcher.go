package dispatcher

import (
	"context"
	"log"
	"time"

	"github.com/llm-operator/job-manager/common/pkg/store"
	ctrl "sigs.k8s.io/controller-runtime"
)

type podCreatorI interface {
	createPod(ctx context.Context, job *store.Job) error
}

func New(
	store *store.S,
	podCreator podCreatorI,
	jobPollingInterval time.Duration,
) *D {
	return &D{
		store:              store,
		podCreator:         podCreator,
		jobPollingInterval: jobPollingInterval,
	}
}

type D struct {
	store      *store.S
	podCreator podCreatorI

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
	log.Printf("Started processing job (ID: %s)\n", job.JobID)
	if err := d.podCreator.createPod(ctx, job); err != nil {
		return err
	}
	return d.store.UpdateJobState(job.JobID, job.Version, store.JobStateRunning)
}
