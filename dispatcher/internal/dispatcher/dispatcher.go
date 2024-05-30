package dispatcher

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	ctrl "sigs.k8s.io/controller-runtime"
)

type jobCreatorI interface {
	createJob(ctx context.Context, job *store.Job, presult *PreProcessResult) error
}

type notebookManagerI interface {
	createNotebook(ctx context.Context, nb *store.Notebook) error
	stopNotebook(ctx context.Context, nb *store.Notebook) error
	deleteNotebook(ctx context.Context, nb *store.Notebook) error
}

// PreProcessorI is an interface for pre-processing jobs.
type PreProcessorI interface {
	Process(ctx context.Context, job *store.Job) (*PreProcessResult, error)
}

// NoopPreProcessor is a no-op implementation of PreProcessorI.
type NoopPreProcessor struct {
}

// Process is a no-op implementation of Process.
func (p *NoopPreProcessor) Process(ctx context.Context, job *store.Job) (*PreProcessResult, error) {
	return &PreProcessResult{}, nil
}

// New returns a new dispatcher.
func New(
	store *store.S,
	jobCreator jobCreatorI,
	preProcessor PreProcessorI,
	nbCreator notebookManagerI,
	pollingInterval time.Duration,
) *D {
	return &D{
		store:           store,
		jobCreator:      jobCreator,
		preProcessor:    preProcessor,
		nbCreator:       nbCreator,
		pollingInterval: pollingInterval,
	}
}

// D is a dispatcher.
type D struct {
	store        *store.S
	jobCreator   jobCreatorI
	preProcessor PreProcessorI
	nbCreator    notebookManagerI

	pollingInterval time.Duration
}

// SetupWithManager registers the dispatcher with the manager.
func (d *D) SetupWithManager(mgr ctrl.Manager) error {
	return mgr.Add(d)
}

// Start starts the dispatcher.
func (d *D) Start(ctx context.Context) error {
	worker := func(initialDelay time.Duration, fn func(context.Context) error) func() error {
		return func() error {
			time.Sleep(initialDelay)
			if err := fn(ctx); err != nil {
				return err
			}
			ticker := time.NewTicker(d.pollingInterval)
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
					if err := fn(ctx); err != nil {
						return err
					}
				}
			}
		}
	}

	maxDelay := time.Second
	g, ctx := errgroup.WithContext(ctx)
	g.Go(worker(time.Duration(rand.Intn(int(maxDelay))), d.processQueuedJobs))
	g.Go(worker(time.Duration(rand.Intn(int(maxDelay))), d.processNotebooks))

	log := ctrl.LoggerFrom(ctx)
	if err := g.Wait(); err != nil {
		log.Error(err, "Run worker")
		return err
	}
	log.Info("Finish dispatcher")
	return nil
}

func (d *D) processQueuedJobs(ctx context.Context) error {
	jobs, err := d.store.ListQueuedJobs()
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

	log.Info("Started pre-processing")
	presult, err := d.preProcessor.Process(ctx, job)
	if err != nil {
		return err
	}
	if err := d.store.UpdateOutputModelID(job.JobID, job.Version, presult.OutputModelID); err != nil {
		return err
	}
	job.Version++
	log.Info("Successfuly completed pre-processing")

	log.Info("Creating a k8s job")
	if err := d.jobCreator.createJob(ctx, job, presult); err != nil {
		return err
	}
	log.Info("Successfully created the k8s job")
	return d.store.UpdateJobState(job.JobID, job.Version, store.JobStateRunning)
}

func (d *D) processNotebooks(ctx context.Context) error {
	nbs, err := d.store.ListQueuedNotebooks()
	if err != nil {
		return err
	}
	for _, nb := range nbs {
		log := ctrl.LoggerFrom(ctx).WithValues("notebookID", nb.NotebookID)
		ctx = ctrl.LoggerInto(ctx, log)

		switch nb.QueuedAction {
		case store.NotebookQueuedActionStart:
			if err := d.startNotebook(ctx, nb); err != nil {
				return err
			}
		case store.NotebookQueuedActionStop:
			if err := d.stopNotebook(ctx, nb); err != nil {
				return err
			}
		case store.NotebookQueuedActionDelete:
			if err := d.deleteNotebook(ctx, nb); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown notebook queued action: %s", nb.QueuedAction)
		}
	}
	return nil
}

func (d *D) startNotebook(ctx context.Context, nb *store.Notebook) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Creating a k8s notebook resources")
	if err := d.nbCreator.createNotebook(ctx, nb); err != nil {
		return err
	}
	log.Info("Successfully created the notebook")

	if err := nb.MutateMessage(func(nb *v1.Notebook) {
		nb.StartedAt = time.Now().UTC().Unix()
		nb.StoppedAt = 0
	}); err != nil {
		return err
	}
	return d.store.SetNonQueuedStateAndMessage(
		nb.NotebookID,
		nb.Version,
		store.NotebookStateRunning,
		nb.Message)
}

func (d *D) stopNotebook(ctx context.Context, nb *store.Notebook) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Stopping a k8s notebook resources")
	if err := d.nbCreator.stopNotebook(ctx, nb); err != nil {
		return err
	}
	log.Info("Successfully stopped the notebook")

	if err := nb.MutateMessage(func(nb *v1.Notebook) {
		nb.StartedAt = 0
		nb.StoppedAt = time.Now().UTC().Unix()
	}); err != nil {
		return err
	}
	return d.store.SetNonQueuedStateAndMessage(
		nb.NotebookID,
		nb.Version,
		store.NotebookStateStopped,
		nb.Message)
}

func (d *D) deleteNotebook(ctx context.Context, nb *store.Notebook) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Deleting a k8s notebook resources")
	if err := d.nbCreator.deleteNotebook(ctx, nb); err != nil {
		return err
	}
	log.Info("Successfully deleted the notebook")
	if err := d.store.DeleteNotebook(nb.NotebookID, nb.ProjectID); err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		log.V(4).Info("Notebook is already deleted")
	}
	return nil
}
