package server

import (
	"context"
	"testing"

	fv1 "github.com/llm-operator/file-manager/api/v1"
	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateJob(t *testing.T) {
	const fileID = "file0"

	tcs := []struct {
		name    string
		req     *v1.CreateJobRequest
		wantErr bool
	}{
		{
			name: "success",
			req: &v1.CreateJobRequest{
				Model:        "model0",
				TrainingFile: fileID,
				Suffix:       "suffix0",
			},
			wantErr: false,
		},
		{
			name: "invalid training file",
			req: &v1.CreateJobRequest{
				Model:        "model0",
				TrainingFile: "invalid file ID",
				Suffix:       "suffix0",
			},
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			st, tearDown := store.NewTest(t)
			defer tearDown()

			srv := New(st, &noopFileGetClient{
				id: fileID,
			})
			resp, err := srv.CreateJob(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			_, err = st.GetJobByJobID(resp.Id)
			assert.NoError(t, err)
		})
	}
}

type noopFileGetClient struct {
	id string
}

func (c *noopFileGetClient) GetFile(ctx context.Context, in *fv1.GetFileRequest, opts ...grpc.CallOption) (*fv1.File, error) {
	if in.Id != c.id {
		return nil, status.Error(codes.NotFound, "file not found")
	}

	return &fv1.File{}, nil
}
