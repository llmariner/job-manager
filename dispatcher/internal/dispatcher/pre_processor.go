package dispatcher

import (
	"context"
	"time"

	fv1 "github.com/llm-operator/file-manager/api/v1"
	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	mv1 "github.com/llm-operator/model-manager/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

const (
	preSignedURLExpire = 7 * 24 * time.Hour
)

type fileClient interface {
	GetFilePath(ctx context.Context, in *fv1.GetFilePathRequest, opts ...grpc.CallOption) (*fv1.GetFilePathResponse, error)
}

type modelClient interface {
	RegisterModel(ctx context.Context, in *mv1.RegisterModelRequest, opts ...grpc.CallOption) (*mv1.RegisterModelResponse, error)
	GetModelPath(ctx context.Context, in *mv1.GetModelPathRequest, opts ...grpc.CallOption) (*mv1.GetModelPathResponse, error)
}

type s3Client interface {
	GeneratePresignedURL(key string, expire time.Duration) (string, error)
}

// NewPreProcessor creates a new pre-processor.
func NewPreProcessor(
	fileClient fileClient,
	modelClient modelClient,
	s3Client s3Client,
) *PreProcessor {
	return &PreProcessor{
		fileClient:  fileClient,
		modelClient: modelClient,
		s3Client:    s3Client,
	}
}

// PreProcessor is a pre-processor.
type PreProcessor struct {
	fileClient  fileClient
	modelClient modelClient
	s3Client    s3Client
}

// PreProcessResult is the result of the pre-process.
type PreProcessResult struct {
	BaseModelURL    string
	TrainingFileURL string

	OutputModelID  string
	OutputModelURL string
}

// Process runs the pre-process.
func (p *PreProcessor) Process(ctx context.Context, job *store.Job) (*PreProcessResult, error) {
	var jobProto v1.Job
	if err := proto.Unmarshal(job.Message, &jobProto); err != nil {
		return nil, err
	}

	mresp, err := p.modelClient.GetModelPath(ctx, &mv1.GetModelPathRequest{
		Id: jobProto.Model,
	})
	if err != nil {
		return nil, err
	}
	baseModelURL, err := p.s3Client.GeneratePresignedURL(mresp.Path, preSignedURLExpire)
	if err != nil {
		return nil, err
	}

	fresp, err := p.fileClient.GetFilePath(ctx, &fv1.GetFilePathRequest{
		Id: jobProto.TrainingFile,
	})
	if err != nil {
		return nil, err
	}
	trainingFileURL, err := p.s3Client.GeneratePresignedURL(fresp.Path, preSignedURLExpire)
	if err != nil {
		return nil, err
	}

	rresp, err := p.modelClient.RegisterModel(ctx, &mv1.RegisterModelRequest{
		BaseModel: jobProto.Model,
		Suffix:    job.Suffix,
		TenantId:  job.TenantID,
	})
	if err != nil {
		return nil, err
	}
	outputModelID := rresp.Id

	outputModelURL, err := p.s3Client.GeneratePresignedURL(rresp.Path, preSignedURLExpire)
	if err != nil {
		return nil, err
	}

	return &PreProcessResult{
		BaseModelURL:    baseModelURL,
		TrainingFileURL: trainingFileURL,

		OutputModelID:  outputModelID,
		OutputModelURL: outputModelURL,
	}, nil
}
