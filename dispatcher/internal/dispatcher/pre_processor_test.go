package dispatcher

import (
	"context"
	"fmt"
	"testing"

	fv1 "github.com/llm-operator/file-manager/api/v1"
	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	mv1 "github.com/llm-operator/model-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

func TestPreProcess(t *testing.T) {
	fc := &fakeFilePathGetter{
		id: "training-file-id",
	}
	mc := &fakeModelPathGetter{
		id: "model-id",
	}

	p := NewPreProcessor(fc, mc)

	jobProto := &v1.Job{
		Model:        "model-id",
		TrainingFile: "training-file-id",
	}
	b, err := proto.Marshal(jobProto)
	assert.NoError(t, err)

	job := &store.Job{
		JobID:   "job-id",
		Message: b,
	}

	err = p.Process(context.Background(), job)
	assert.NoError(t, err)
}

type fakeFilePathGetter struct {
	id string
}

func (f *fakeFilePathGetter) GetFilePath(ctx context.Context, in *fv1.GetFilePathRequest, opts ...grpc.CallOption) (*fv1.GetFilePathResponse, error) {
	if in.Id != f.id {
		return nil, fmt.Errorf("unexpected id: %s", in.Id)
	}

	return &fv1.GetFilePathResponse{
		Path: "fakeFilePath",
	}, nil
}

type fakeModelPathGetter struct {
	id string
}

func (f *fakeModelPathGetter) GetModelPath(ctx context.Context, in *mv1.GetModelPathRequest, opts ...grpc.CallOption) (*mv1.GetModelPathResponse, error) {
	if in.Id != f.id {
		return nil, fmt.Errorf("unexpected id: %s", in.Id)
	}

	return &mv1.GetModelPathResponse{
		Path: "fakeModelPath",
	}, nil
}
