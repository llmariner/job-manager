package dispatcher

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	fv1 "github.com/llm-operator/file-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	is3 "github.com/llm-operator/job-manager/dispatcher/internal/s3"
	mv1 "github.com/llm-operator/model-manager/api/v1"
	"google.golang.org/grpc"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	preSignedURLExpire = 7 * 24 * time.Hour
)

type fileClient interface {
	GetFilePath(ctx context.Context, in *fv1.GetFilePathRequest, opts ...grpc.CallOption) (*fv1.GetFilePathResponse, error)
}

type modelClient interface {
	RegisterModel(ctx context.Context, in *mv1.RegisterModelRequest, opts ...grpc.CallOption) (*mv1.RegisterModelResponse, error)
	GetBaseModelPath(ctx context.Context, in *mv1.GetBaseModelPathRequest, opts ...grpc.CallOption) (*mv1.GetBaseModelPathResponse, error)
}

type s3Client interface {
	GeneratePresignedURL(key string, expire time.Duration, requestType is3.RequestType) (string, error)
	ListObjectsPages(prefix string, f func(page *s3.ListObjectsOutput, lastPage bool) bool) error
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
	BaseModelURLs   map[string]string
	TrainingFileURL string

	OutputModelID  string
	OutputModelURL string
}

// Process runs the pre-process.
func (p *PreProcessor) Process(ctx context.Context, job *store.Job) (*PreProcessResult, error) {
	log := ctrl.LoggerFrom(ctx)

	jobProto, err := job.V1Job()
	if err != nil {
		return nil, err
	}

	mresp, err := p.modelClient.GetBaseModelPath(ctx, &mv1.GetBaseModelPathRequest{
		Id: jobProto.Model,
	})
	if err != nil {
		return nil, fmt.Errorf("get base model path: %s", err)
	}
	// Find all files under the path and generate pre-signed URLs for all of them.
	var paths []string
	f := func(page *s3.ListObjectsOutput, lastPage bool) bool {
		for _, obj := range page.Contents {
			paths = append(paths, *obj.Key)
		}
		return lastPage
	}
	if err := p.s3Client.ListObjectsPages(mresp.Path, f); err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no files found under %s", mresp.Path)
	}
	log.Info(fmt.Sprintf("Found %d objects under %s", len(paths), mresp.Path))

	baseModelURLs := map[string]string{}
	for _, path := range paths {
		url, err := p.s3Client.GeneratePresignedURL(path, preSignedURLExpire, is3.RequestTypeGetObject)
		if err != nil {
			return nil, fmt.Errorf("generate presigned url: %s", err)
		}
		baseModelURLs[strings.TrimPrefix(path, mresp.Path+"/")] = url
	}

	fresp, err := p.fileClient.GetFilePath(ctx, &fv1.GetFilePathRequest{
		Id: jobProto.TrainingFile,
	})
	if err != nil {
		return nil, fmt.Errorf("get file path: %s", err)
	}
	trainingFileURL, err := p.s3Client.GeneratePresignedURL(fresp.Path, preSignedURLExpire, is3.RequestTypeGetObject)
	if err != nil {
		return nil, fmt.Errorf("generate presigned url: %s", err)
	}

	rresp, err := p.modelClient.RegisterModel(ctx, &mv1.RegisterModelRequest{
		BaseModel: jobProto.Model,
		Suffix:    job.Suffix,
		TenantId:  job.TenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("register model: %s", err)
	}
	outputModelID := rresp.Id

	outputModelURL, err := p.s3Client.GeneratePresignedURL(rresp.Path, preSignedURLExpire, is3.RequestTypePutObject)
	if err != nil {
		return nil, fmt.Errorf("generate presigned url: %s", err)
	}

	return &PreProcessResult{
		BaseModelURLs:   baseModelURLs,
		TrainingFileURL: trainingFileURL,
		OutputModelID:   outputModelID,
		OutputModelURL:  outputModelURL,
	}, nil
}
