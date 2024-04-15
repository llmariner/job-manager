package dispatcher

import (
	"context"

	fv1 "github.com/llm-operator/file-manager/api/v1"
	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	mv1 "github.com/llm-operator/model-manager/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type filePathGetterClient interface {
	GetFilePath(ctx context.Context, in *fv1.GetFilePathRequest, opts ...grpc.CallOption) (*fv1.GetFilePathResponse, error)
}

type modelPathGetterClient interface {
	GetModelPath(ctx context.Context, in *mv1.GetModelPathRequest, opts ...grpc.CallOption) (*mv1.GetModelPathResponse, error)
}

// NewPreProcessor creates a new pre-processor.
func NewPreProcessor(
	filePathGetterClient filePathGetterClient,
	modelPathGetterClient modelPathGetterClient,
) *PreProcessor {
	return &PreProcessor{
		filePathGetterClient:  filePathGetterClient,
		modelPathGetterClient: modelPathGetterClient,
	}
}

// PreProcessor is a pre-processor.
type PreProcessor struct {
	filePathGetterClient  filePathGetterClient
	modelPathGetterClient modelPathGetterClient
}

// Process runs the pre-process.
func (p *PreProcessor) Process(ctx context.Context, job *store.Job) error {
	var jobProto v1.Job
	if err := proto.Unmarshal(job.Message, &jobProto); err != nil {
		return err
	}

	if _, err := p.filePathGetterClient.GetFilePath(ctx, &fv1.GetFilePathRequest{
		Id: jobProto.TrainingFile,
	}); err != nil {
		return err
	}

	if _, err := p.modelPathGetterClient.GetModelPath(ctx, &mv1.GetModelPathRequest{
		Id: jobProto.Model,
	}); err != nil {
		return err
	}

	// TODO(kenji): Download the training file and model.

	return nil
}
