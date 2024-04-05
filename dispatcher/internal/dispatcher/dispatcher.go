package dispatcher

import (
	"context"
	"log"
	"time"

	"github.com/llm-operator/job-manager/common/pkg/store"
)

type podCreatorI interface {
	createPod(ctx context.Context, job *store.Job) error
}

func New(
	store *store.S,
	podCreator podCreatorI,
) *D {
	return &D{
		store:      store,
		podCreator: podCreator,
	}
}

type D struct {
	store      *store.S
	podCreator podCreatorI
}

// Run runs the dispatcher.
func (d *D) Run(ctx context.Context, interval time.Duration) error {
	// TODO(kenji): Reconcile running jobs.
	// This is necessary to handle the case where the dispatcher is restarted or the informer misses some events.

	if err := d.processPendingJobs(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(interval)
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
	if err := d.store.UpdateJobState(job.JobID, job.Version, store.JobStateRunning); err != nil {
		return err
	}
	job.Version++

	if err := d.podCreator.createPod(ctx, job); err != nil {
		return err
	}

	// TODO(kenji): Watch pods and update job state. The code should be changed to check the completion of the job in an async fashion.

	log.Printf("Finished processing job (ID: %s)\n", job.JobID)
	if err := d.store.UpdateJobState(job.JobID, job.Version, store.JobStateCompleted); err != nil {
		return err
	}
	return nil
}
