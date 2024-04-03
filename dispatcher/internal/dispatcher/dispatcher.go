package dispatcher

import (
	"context"
	"log"
	"time"

	"github.com/llm-operator/job-manager/common/pkg/store"
)

func New(store *store.S) *D {
	return &D{
		store: store,
	}
}

type D struct {
	store *store.S
}

// Run runs the dispatcher.
func (d *D) Run(ctx context.Context, interval time.Duration) error {
	// TODO(kenji): If there is any running job, restore its state.

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
	log.Printf("Processing job (ID: %s)\n", job.JobID)
	if err := d.store.UpdateJobState(job.JobID, job.Version, store.JobStateRunning); err != nil {
		return err
	}
	job.Version++

	// Update the state to completed.

	if err := d.store.UpdateJobState(job.JobID, job.Version, store.JobStateCompleted); err != nil {
		return err
	}
	return nil
}
