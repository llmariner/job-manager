package dispatcher

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	v1 "github.com/llm-operator/job-manager/api/v1"
	is3 "github.com/llm-operator/job-manager/dispatcher/internal/s3"
	fv1 "github.com/llmariner/file-manager/api/v1"
	mv1 "github.com/llmariner/model-manager/api/v1"
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
	GeneratePresignedURL(ctx context.Context, key string, expire time.Duration, requestType is3.RequestType) (string, error)
	ListObjectsPages(ctx context.Context, prefix string) (*s3.ListObjectsV2Output, error)
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
	BaseModelURLs     map[string]string
	TrainingFileURL   string
	ValidationFileURL string

	OutputModelID  string
	OutputModelURL string
}

// Process runs the pre-process.
func (p *PreProcessor) Process(ctx context.Context, job *v1.InternalJob) (*PreProcessResult, error) {
	log := ctrl.LoggerFrom(ctx)

	mresp, err := p.modelClient.GetBaseModelPath(ctx, &mv1.GetBaseModelPathRequest{
		Id: job.Job.Model,
	})
	if err != nil {
		return nil, fmt.Errorf("get base model path: %s", err)
	}
	// Find all files under the path and generate pre-signed URLs for all of them.
	var paths []string
	for {
		result, err := p.s3Client.ListObjectsPages(ctx, mresp.Path)
		if err != nil {
			return nil, err
		}
		for _, obj := range result.Contents {
			paths = append(paths, *obj.Key)
		}
		if !*result.IsTruncated {
			break
		}
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("no files found under %s", mresp.Path)
	}
	log.Info(fmt.Sprintf("Found %d objects under %s", len(paths), mresp.Path))

	baseModelURLs := map[string]string{}
	for _, path := range paths {
		url, err := p.s3Client.GeneratePresignedURL(ctx, path, preSignedURLExpire, is3.RequestTypeGetObject)
		if err != nil {
			return nil, fmt.Errorf("generate presigned url: %s", err)
		}
		baseModelURLs[strings.TrimPrefix(path, mresp.Path+"/")] = url
	}

	trainingFileURL, err := p.getPresignedURLForFile(ctx, job.Job.TrainingFile)
	if err != nil {
		return nil, err
	}
	var validationFileURL string
	if f := job.Job.ValidationFile; f != "" {
		validationFileURL, err = p.getPresignedURLForFile(ctx, f)
		if err != nil {
			return nil, err
		}
	}

	rresp, err := p.modelClient.RegisterModel(ctx, &mv1.RegisterModelRequest{
		BaseModel:      job.Job.Model,
		Suffix:         job.Suffix,
		OrganizationId: job.Job.OrganizationId,
		ProjectId:      job.Job.ProjectId,
	})
	if err != nil {
		return nil, fmt.Errorf("register model: %s", err)
	}
	outputModelID := rresp.Id

	outputModelURL, err := p.s3Client.GeneratePresignedURL(ctx, rresp.Path, preSignedURLExpire, is3.RequestTypePutObject)
	if err != nil {
		return nil, fmt.Errorf("generate presigned url: %s", err)
	}

	return &PreProcessResult{
		BaseModelURLs:     baseModelURLs,
		TrainingFileURL:   trainingFileURL,
		ValidationFileURL: validationFileURL,
		OutputModelID:     outputModelID,
		OutputModelURL:    outputModelURL,
	}, nil
}

func (p *PreProcessor) getPresignedURLForFile(ctx context.Context, fileID string) (string, error) {
	fresp, err := p.fileClient.GetFilePath(ctx, &fv1.GetFilePathRequest{
		Id: fileID,
	})
	if err != nil {
		return "", fmt.Errorf("get file path: %s", err)
	}
	url, err := p.s3Client.GeneratePresignedURL(ctx, fresp.Path, preSignedURLExpire, is3.RequestTypeGetObject)
	if err != nil {
		return "", fmt.Errorf("generate presigned url: %s", err)
	}
	return url, nil
}
