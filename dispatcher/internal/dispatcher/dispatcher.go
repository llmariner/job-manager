package dispatcher

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/rbac-manager/pkg/auth"
	"golang.org/x/sync/errgroup"
	ctrl "sigs.k8s.io/controller-runtime"
)

type jobCreatorI interface {
	createJob(ctx context.Context, job *v1.InternalJob, presult *PreProcessResult) error
}

type notebookManagerI interface {
	createNotebook(ctx context.Context, nb *v1.InternalNotebook) error
	stopNotebook(ctx context.Context, nb *v1.InternalNotebook) error
	deleteNotebook(ctx context.Context, nb *v1.InternalNotebook) error
}

type batchJobManagerI interface {
	createBatchJob(ctx context.Context, job *v1.InternalBatchJob) error
	cancelBatchJob(ctx context.Context, job *v1.InternalBatchJob) error
}

// PreProcessorI is an interface for pre-processing jobs.
type PreProcessorI interface {
	Process(ctx context.Context, job *v1.InternalJob) (*PreProcessResult, error)
}

// NoopPreProcessor is a no-op implementation of PreProcessorI.
type NoopPreProcessor struct {
}

// Process is a no-op implementation of Process.
func (p *NoopPreProcessor) Process(ctx context.Context, job *v1.InternalJob) (*PreProcessResult, error) {
	return &PreProcessResult{}, nil
}

// New returns a new dispatcher.
func New(
	ftClient v1.FineTuningWorkerServiceClient,
	wsClient v1.WorkspaceWorkerServiceClient,
	bwClient v1.BatchWorkerServiceClient,
	jobCreator jobCreatorI,
	preProcessor PreProcessorI,
	nbCreator notebookManagerI,
	bjManager batchJobManagerI,
	pollingInterval time.Duration,
) *D {
	return &D{
		ftClient:        ftClient,
		wsClient:        wsClient,
		bwClient:        bwClient,
		jobCreator:      jobCreator,
		preProcessor:    preProcessor,
		nbCreator:       nbCreator,
		bjManager:       bjManager,
		pollingInterval: pollingInterval,
	}
}

// D is a dispatcher.
type D struct {
	ftClient v1.FineTuningWorkerServiceClient
	wsClient v1.WorkspaceWorkerServiceClient
	bwClient v1.BatchWorkerServiceClient

	jobCreator   jobCreatorI
	preProcessor PreProcessorI
	nbCreator    notebookManagerI
	bjManager    batchJobManagerI

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
	g.Go(worker(time.Duration(rand.Intn(int(maxDelay))), d.processBatchJobs))

	log := ctrl.LoggerFrom(ctx)
	if err := g.Wait(); err != nil {
		log.Error(err, "Run worker")
		return err
	}
	log.Info("Finish dispatcher")
	return nil
}

func (d *D) processQueuedJobs(ctx context.Context) error {
	ctx = auth.AppendWorkerAuthorization(ctx)
	resp, err := d.ftClient.ListQueuedInternalJobs(ctx, &v1.ListQueuedInternalJobsRequest{})
	if err != nil {
		return err
	}

	for _, job := range resp.Jobs {
		if err := d.processJob(ctx, job); err != nil {
			return err
		}
	}
	return nil
}

func (d *D) processJob(ctx context.Context, job *v1.InternalJob) error {
	log := ctrl.LoggerFrom(ctx).WithValues("jobID", job.Job.Id)
	log.Info("Processing job")

	ctx = ctrl.LoggerInto(ctx, log)

	log.Info("Started pre-processing")
	presult, err := d.preProcessor.Process(ctx, job)
	if err != nil {
		return err
	}
	if _, err := d.ftClient.UpdateJobPhase(ctx, &v1.UpdateJobPhaseRequest{
		Id:      job.Job.Id,
		Phase:   v1.UpdateJobPhaseRequest_PREPROCESSED,
		ModelId: presult.OutputModelID,
	}); err != nil {
		return err
	}
	log.Info("Successfuly completed pre-processing")

	log.Info("Creating a k8s job")
	if err := d.jobCreator.createJob(ctx, job, presult); err != nil {
		return err
	}
	log.Info("Successfully created the k8s job")
	_, err = d.ftClient.UpdateJobPhase(ctx, &v1.UpdateJobPhaseRequest{
		Id:    job.Job.Id,
		Phase: v1.UpdateJobPhaseRequest_JOB_CREATED,
	})
	return err
}

func (d *D) processNotebooks(ctx context.Context) error {
	ctx = auth.AppendWorkerAuthorization(ctx)
	resp, err := d.wsClient.ListQueuedInternalNotebooks(ctx, &v1.ListQueuedInternalNotebooksRequest{})
	if err != nil {
		return err
	}
	for _, nb := range resp.Notebooks {
		log := ctrl.LoggerFrom(ctx).WithValues("notebookID", nb.Notebook.Id)
		ctx = ctrl.LoggerInto(ctx, log)

		var (
			state v1.NotebookState
			err   error
		)
		switch nb.QueuedAction {
		case v1.NotebookQueuedAction_STARTING:
			log.Info("Creating a k8s notebook resources")
			err = d.nbCreator.createNotebook(ctx, nb)
			state = v1.NotebookState_INITIALIZING
		case v1.NotebookQueuedAction_STOPPING:
			log.Info("Stopping a k8s notebook resources")
			err = d.nbCreator.stopNotebook(ctx, nb)
			state = v1.NotebookState_STOPPED
		case v1.NotebookQueuedAction_DELETING:
			log.Info("Deleting a k8s notebook resources")
			err = d.nbCreator.deleteNotebook(ctx, nb)
			state = v1.NotebookState_DELETED
		case v1.NotebookQueuedAction_ACTION_UNSPECIFIED:
			return fmt.Errorf("notebook queued action is not specified")
		default:
			return fmt.Errorf("unknown notebook queued action: %s", nb.QueuedAction)
		}
		if err != nil {
			return fmt.Errorf("failed to %s the notebook: %s", nb.QueuedAction.String(), err)
		}
		log.Info("Successfully completed the action", "action", nb.QueuedAction.String())

		if _, err := d.wsClient.UpdateNotebookState(ctx, &v1.UpdateNotebookStateRequest{
			Id:    nb.Notebook.Id,
			State: state,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (d *D) processBatchJobs(ctx context.Context) error {
	ctx = auth.AppendWorkerAuthorization(ctx)
	resp, err := d.bwClient.ListQueuedInternalBatchJobs(ctx, &v1.ListQueuedInternalBatchJobsRequest{})
	if err != nil {
		return err
	}
	for _, job := range resp.Jobs {
		log := ctrl.LoggerFrom(ctx).WithValues("batchJobID", job.Job.Id)
		ctx = ctrl.LoggerInto(ctx, log)

		var (
			state v1.InternalBatchJob_State
			err   error
		)
		switch job.QueuedAction {
		case v1.InternalBatchJob_CREATING:
			log.Info("Creating a new batch job")
			err = d.bjManager.createBatchJob(ctx, job)
			state = v1.InternalBatchJob_RUNNING
		case v1.InternalBatchJob_CANCELING:
			log.Info("Canceling a batch job")
			err = d.bjManager.cancelBatchJob(ctx, job)
			state = v1.InternalBatchJob_CANCELED
		case v1.InternalBatchJob_ACTION_UNSPECIFIED:
			return fmt.Errorf("batch job queued action is not specified")
		default:
			return fmt.Errorf("unknown batch job queued action: %s", job.QueuedAction)
		}
		if err != nil {
			return fmt.Errorf("failed to %s the batch job: %s", job.QueuedAction, err)
		}
		log.Info("Successfully completed the action", "action", job.QueuedAction.String())

		if _, err := d.bwClient.UpdateBatchJobState(ctx, &v1.UpdateBatchJobStateRequest{
			Id:    job.Job.Id,
			State: state,
		}); err != nil {
			return err
		}
	}
	return nil
}
