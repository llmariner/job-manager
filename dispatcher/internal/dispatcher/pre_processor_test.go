package dispatcher

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	fv1 "github.com/llmariner/file-manager/api/v1"
	v1 "github.com/llmariner/job-manager/api/v1"
	is3 "github.com/llmariner/job-manager/dispatcher/internal/s3"
	mv1 "github.com/llmariner/model-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

func TestPreProcess(t *testing.T) {
	fc := &fakeFileClient{
		ids: map[string]string{
			"training-file-id":   "training-file-path",
			"validation-file-id": "validation-file-path",
		},
	}
	mc := &fakeModelClient{
		id: "model-id",
	}
	sc := &fakeS3Client{}

	p := NewPreProcessor(fc, mc, sc)

	job := &v1.InternalJob{
		Job: &v1.Job{
			Id:             "job-id",
			Model:          "model-id",
			TrainingFile:   "training-file-id",
			ValidationFile: "validation-file-id",
		},
	}

	got, err := p.Process(context.Background(), job)
	assert.NoError(t, err)
	want := &PreProcessResult{
		BaseModelURLs: map[string]string{
			"obj1":      "presigned-model-path/obj1",
			"path/obj2": "presigned-model-path/path/obj2",
		},
		TrainingFileURL:         "presigned-training-file-path",
		ValidationFileURL:       "presigned-validation-file-path",
		OutputModelID:           "generated-model-id",
		OutputModelURL:          "http://example.com",
		OutputModelPresignFlags: "-F 'key0=value0'",
	}
	assert.Equal(t, want, got)
}

type fakeFileClient struct {
	ids map[string]string
}

func (f *fakeFileClient) GetFilePath(ctx context.Context, in *fv1.GetFilePathRequest, opts ...grpc.CallOption) (*fv1.GetFilePathResponse, error) {
	p, ok := f.ids[in.Id]
	if !ok {
		return nil, fmt.Errorf("unexpected id: %s", in.Id)
	}

	return &fv1.GetFilePathResponse{
		Path: p,
	}, nil
}

type fakeModelClient struct {
	id string
}

func (f *fakeModelClient) RegisterModel(ctx context.Context, in *mv1.RegisterModelRequest, opts ...grpc.CallOption) (*mv1.RegisterModelResponse, error) {
	return &mv1.RegisterModelResponse{
		Id:   "generated-model-id",
		Path: "generated-model-path",
	}, nil
}

func (f *fakeModelClient) GetBaseModelPath(ctx context.Context, in *mv1.GetBaseModelPathRequest, opts ...grpc.CallOption) (*mv1.GetBaseModelPathResponse, error) {
	if in.Id != f.id {
		return nil, fmt.Errorf("unexpected id: %s", in.Id)
	}

	return &mv1.GetBaseModelPathResponse{
		Path: "model-path",
	}, nil
}

type fakeS3Client struct {
}

func (c *fakeS3Client) GeneratePresignedURL(ctx context.Context, key string, expire time.Duration, requestType is3.RequestType) (string, error) {
	return fmt.Sprintf("presigned-%s", key), nil
}

func (c *fakeS3Client) GeneratePresignedURLForPost(ctx context.Context, keyPrefix string, expire time.Duration) (*s3.PresignedPostRequest, error) {
	return &s3.PresignedPostRequest{
		URL: "http://example.com",
		Values: map[string]string{
			"key0": "value0",
		},
	}, nil
}

func (c *fakeS3Client) ListObjectsPages(ctx context.Context, prefix string) (*s3.ListObjectsV2Output, error) {
	return &s3.ListObjectsV2Output{
		Contents: []types.Object{
			{Key: proto.String("model-path/obj1")},
			{Key: proto.String("model-path/path/obj2")},
		},
		IsTruncated: proto.Bool(false),
	}, nil
}
