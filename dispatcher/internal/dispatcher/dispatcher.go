package dispatcher

import (
	"context"
	"log"
	"time"

	iv1 "github.com/llm-operator/inference-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	"google.golang.org/grpc"
)

type podCreatorI interface {
	createPod(ctx context.Context, job *store.Job) error
}

// ModelRegisterClient is the client for the model register service.
type ModelRegisterClient interface {
	RegisterModel(ctx context.Context, in *iv1.RegisterModelRequest, opts ...grpc.CallOption) (*iv1.RegisterModelResponse, error)
}

// NoopModelRegisterClient is a no-op implementation of ModelRegisterClient.
type NoopModelRegisterClient struct {
}

func (c *NoopModelRegisterClient) RegisterModel(ctx context.Context, in *iv1.RegisterModelRequest, opts ...grpc.CallOption) (*iv1.RegisterModelResponse, error) {
	return &iv1.RegisterModelResponse{}, nil
}

func New(
	store *store.S,
	podCreator podCreatorI,
	modelRegisterClient ModelRegisterClient,
) *D {
	return &D{
		store:               store,
		podCreator:          podCreator,
		modelRegisterClient: modelRegisterClient,
	}
}

type D struct {
	store               *store.S
	podCreator          podCreatorI
	modelRegisterClient ModelRegisterClient
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

	if _, err := d.modelRegisterClient.RegisterModel(ctx, &iv1.RegisterModelRequest{
		// TODO(kenji): Fix this.
		ModelName:   "gemma:2b-fine-tuned",
		BaseModel:   "gemma:2b",
		AdapterPath: "/adapter/ggml-adapter-model.bin",
	}); err != nil {
		return err
	}

	log.Printf("Finished processing job (ID: %s)\n", job.JobID)
	if err := d.store.UpdateJobState(job.JobID, job.Version, store.JobStateCompleted); err != nil {
		return err
	}
	return nil
}
