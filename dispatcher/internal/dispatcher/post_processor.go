package dispatcher

import (
	"context"
	"fmt"

	"github.com/llm-operator/job-manager/common/pkg/store"
	mv1 "github.com/llm-operator/model-manager/api/v1"
	"google.golang.org/grpc"
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
func (p *PostProcessor) Process(ctx context.Context, job *store.Job) error {
	if job.OutputModelID == "" {
		return fmt.Errorf("output model ID is not populated")
	}

	if _, err := p.modelClient.PublishModel(ctx, &mv1.PublishModelRequest{
		Id:       job.OutputModelID,
		TenantId: job.TenantID,
	}); err != nil {
		return err
	}

	return nil

}
