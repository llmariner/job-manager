package dispatcher

import (
	"context"
	"fmt"
	"testing"

	v1 "github.com/llmariner/job-manager/api/v1"
	mv1 "github.com/llmariner/model-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestPostProcessor(t *testing.T) {
	mc := &fakeModelPublishClient{
		expectedID: "output-model-id",
	}

	p := NewPostProcessor(mc)
	job := &v1.InternalJob{
		Job:           &v1.Job{Id: "job-id"},
		OutputModelId: "output-model-id",
	}

	err := p.Process(context.Background(), job)
	assert.NoError(t, err)
}

type fakeModelPublishClient struct {
	expectedID string
}

func (f *fakeModelPublishClient) PublishModel(ctx context.Context, in *mv1.PublishModelRequest, opts ...grpc.CallOption) (*mv1.PublishModelResponse, error) {
	if in.Id != f.expectedID {
		return nil, fmt.Errorf("unexpected model id: %s", in.Id)
	}
	return &mv1.PublishModelResponse{}, nil
}
