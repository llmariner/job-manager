package dispatcher

import (
	"context"
	"fmt"

	v1 "github.com/llm-operator/job-manager/api/v1"
	mv1 "github.com/llmariner/model-manager/api/v1"
	"google.golang.org/grpc"
	ctrl "sigs.k8s.io/controller-runtime"
)

type modelPublishClient interface {
	PublishModel(ctx context.Context, in *mv1.PublishModelRequest, opts ...grpc.CallOption) (*mv1.PublishModelResponse, error)
}

// NewPostProcessor creates a new PostProcessor.
func NewPostProcessor(
	modelClient modelPublishClient,
) *PostProcessor {
	return &PostProcessor{
		modelClient: modelClient,
	}
}

// PostProcessor is a post-processor.
type PostProcessor struct {
	modelClient modelPublishClient
}

// Process processes the job.
func (p *PostProcessor) Process(ctx context.Context, job *v1.InternalJob) error {
	log := ctrl.LoggerFrom(ctx)

	if job.OutputModelId == "" {
		return fmt.Errorf("output model ID is not populated")
	}

	log.Info("Publishing the model")
	if _, err := p.modelClient.PublishModel(ctx, &mv1.PublishModelRequest{
		Id: job.OutputModelId,
	}); err != nil {
		return err
	}

	return nil

}
