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
			}, nil)
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

func TestJobCancel(t *testing.T) {
	const jobID = "job-1"
	var tcs = []struct {
		name  string
		state store.JobState
		want  *v1.Job
	}{
		{
			name:  "transit pending to cancelled",
			state: store.JobStateQueued,
			want:  &v1.Job{Status: string(store.JobStateCancelled)},
		},
		{
			name:  "transit running to cancelled",
			state: store.JobStateRunning,
			want:  &v1.Job{Status: string(store.JobStateCancelled)},
		},
		{
			name:  "keep completed state",
			state: store.JobStateSucceeded,
			want:  &v1.Job{Status: string(store.JobStateSucceeded)},
		},
		{
			name:  "keep cancelled state",
			state: store.JobStateCancelled,
			want:  &v1.Job{Status: string(store.JobStateCancelled)},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			st, tearDown := store.NewTest(t)
			defer tearDown()

			err := st.CreateJob(&store.Job{JobID: jobID, State: tc.state})
			assert.NoError(t, err)

			srv := New(st, nil, &noopK8sJobClient{})
			resp, err := srv.CancelJob(context.Background(), &v1.CancelJobRequest{Id: jobID})
			assert.NoError(t, err)
			assert.Equal(t, tc.want.Status, resp.Status)
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

type noopK8sJobClient struct{}

func (c *noopK8sJobClient) CancelJob(ctx context.Context, jobID string) error {
	return nil
}
