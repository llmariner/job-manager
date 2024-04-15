package dispatcher

import (
	"context"
	"io"
	"os"

	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	mv1 "github.com/llm-operator/model-manager/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type modelCreatorClient interface {
	RegisterModel(ctx context.Context, in *mv1.RegisterModelRequest, opts ...grpc.CallOption) (*mv1.RegisterModelResponse, error)
	PublishModel(ctx context.Context, in *mv1.PublishModelRequest, opts ...grpc.CallOption) (*mv1.PublishModelResponse, error)
}

// s3Client is an interface for an S3 client.
type s3Client interface {
	Upload(r io.Reader, key string) error
}

// NewPostProcessor creates a new PostProcessor.
func NewPostProcessor(
	modelCreatorClient modelCreatorClient,
	s3Client s3Client,
) *PostProcessor {
	return &PostProcessor{
		modelCreatorClient: modelCreatorClient,
		s3Client:           s3Client,
	}
}

// PostProcessor is a post-processor.
type PostProcessor struct {
	modelCreatorClient modelCreatorClient
	s3Client           s3Client
}

// Process processes the job.
func (p *PostProcessor) Process(ctx context.Context, job *store.Job) error {
	var jobProto v1.Job
	if err := proto.Unmarshal(job.Message, &jobProto); err != nil {
		return err
	}
	resp, err := p.modelCreatorClient.RegisterModel(ctx, &mv1.RegisterModelRequest{
		BaseModel: jobProto.Model,
		Suffix:    job.Suffix,
		TenantId:  job.TenantID,
	})
	if err != nil {
		return err
	}

	// TODO(kenji): Provide a unique location per model. Or make the pod just upload the model directly.
	r, err := os.Open("/models/adapter/ggml-adapter-model.bin")
	if err != nil {
		return err
	}
	if err := p.s3Client.Upload(r, resp.Path); err != nil {
		return err
	}

	if _, err := p.modelCreatorClient.PublishModel(ctx, &mv1.PublishModelRequest{
		Id:       resp.Id,
		TenantId: job.TenantID,
	}); err != nil {
		return err
	}

	return nil

}
