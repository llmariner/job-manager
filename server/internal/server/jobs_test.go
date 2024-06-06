package server

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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestCreateJob(t *testing.T) {
	const (
		tFileID = "tFile0"
		vFileID = "vFile0"
		modelID = "model0"
	)

	tcs := []struct {
		name    string
		req     *v1.CreateJobRequest
		wantErr bool
	}{
		{
			name: "success",
			req: &v1.CreateJobRequest{
				Model:        modelID,
				TrainingFile: tFileID,
				Suffix:       "suffix0",
			},
			wantErr: false,
		},
		{
			name: "success with validation file",
			req: &v1.CreateJobRequest{
				Model:          modelID,
				TrainingFile:   tFileID,
				ValidationFile: vFileID,
				Suffix:         "suffix0",
			},
			wantErr: false,
		},
		{
			name: "invalid training file",
			req: &v1.CreateJobRequest{
				Model:        modelID,
				TrainingFile: "invalid file ID",
				Suffix:       "suffix0",
			},
			wantErr: true,
		},
		{
			name: "invalid model",
			req: &v1.CreateJobRequest{
				Model:        "invalid model ID",
				TrainingFile: tFileID,
				Suffix:       "suffix0",
			},
			wantErr: true,
		},
		{
			name: "invalida validation file",
			req: &v1.CreateJobRequest{
				Model:          modelID,
				TrainingFile:   tFileID,
				ValidationFile: "invalid file ID",
				Suffix:         "suffix0",
			},
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			st, tearDown := store.NewTest(t)
			defer tearDown()

			srv := New(
				st,
				&noopFileGetClient{
					ids: map[string]bool{
						tFileID: true,
						vFileID: true,
					},
				},
				&noopModelClient{
					id: modelID,
				},
				nil,
				nil)
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("Authorization", "dummy"))
			resp, err := srv.CreateJob(ctx, tc.req)
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

func TestListJobs(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	for i := 0; i < 10; i++ {
		jobProto := &v1.Job{
			Id: fmt.Sprintf("job%d", i),
		}
		msg, err := proto.Marshal(jobProto)
		assert.NoError(t, err)
		job := &store.Job{
			JobID:     jobProto.Id,
			Message:   msg,
			TenantID:  defaultTenantID,
			ProjectID: defaultProjectID,
		}
		err = st.CreateJob(job)
		assert.NoError(t, err)
	}

	srv := New(st, nil, nil, &noopK8sClient{}, nil)
	resp, err := srv.ListJobs(context.Background(), &v1.ListJobsRequest{Limit: 5})
	assert.NoError(t, err)
	assert.True(t, resp.HasMore)
	assert.Len(t, resp.Data, 5)
	want := []string{"job9", "job8", "job7", "job6", "job5"}
	for i, job := range resp.Data {
		assert.Equal(t, want[i], job.Id)
	}

	resp, err = srv.ListJobs(context.Background(), &v1.ListJobsRequest{After: resp.Data[4].Id, Limit: 2})
	assert.NoError(t, err)
	assert.True(t, resp.HasMore)
	assert.Len(t, resp.Data, 2)
	want = []string{"job4", "job3"}
	for i, job := range resp.Data {
		assert.Equal(t, want[i], job.Id)
	}

	resp, err = srv.ListJobs(context.Background(), &v1.ListJobsRequest{After: resp.Data[1].Id, Limit: 3})
	assert.NoError(t, err)
	assert.False(t, resp.HasMore)
	assert.Len(t, resp.Data, 3)
	want = []string{"job2", "job1", "job0"}
	for i, job := range resp.Data {
		assert.Equal(t, want[i], job.Id)
	}
}

func TestGetJob(t *testing.T) {
	const jobID = "job-1"

	st, tearDown := store.NewTest(t)
	defer tearDown()

	err := st.CreateJob(&store.Job{
		JobID:     jobID,
		TenantID:  defaultTenantID,
		ProjectID: defaultProjectID,
		State:     store.JobStateQueued,
	})
	assert.NoError(t, err)

	srv := New(st, nil, nil, &noopK8sClient{}, nil)
	resp, err := srv.GetJob(context.Background(), &v1.GetJobRequest{Id: jobID})
	assert.NoError(t, err)
	assert.Equal(t, store.JobStateQueued, store.JobState(resp.Status))
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

			err := st.CreateJob(&store.Job{JobID: jobID, State: tc.state, TenantID: defaultTenantID, ProjectID: defaultProjectID})
			assert.NoError(t, err)

			srv := New(st, nil, nil, &noopK8sClient{}, nil)
			resp, err := srv.CancelJob(context.Background(), &v1.CancelJobRequest{Id: jobID})
			assert.NoError(t, err)
			assert.Equal(t, tc.want.Status, resp.Status)
		})
	}
}

func TestListQueuedInternalJobs(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	jobs := []*store.Job{
		{
			State:    store.JobStateQueued,
			TenantID: "t0",
		},
		{
			State:    store.JobStateRunning,
			TenantID: "t0",
		},
		{
			State:    store.JobStateQueued,
			TenantID: "t1",
		},
		{
			State:    store.JobStateQueued,
			TenantID: "t0",
		},
	}
	for i, job := range jobs {
		jobProto := &v1.Job{
			Id: fmt.Sprintf("job%d", i),
		}
		msg, err := proto.Marshal(jobProto)
		assert.NoError(t, err)
		assert.NoError(t, st.CreateJob(&store.Job{
			JobID:    jobProto.Id,
			State:    job.State,
			Message:  msg,
			TenantID: job.TenantID,
		}))
	}

	srv := NewWorkerServiceServer(st)
	req := &v1.ListQueuedInternalJobsRequest{TenantId: "t0"}
	got, err := srv.ListQueuedInternalJobs(context.Background(), req)
	assert.NoError(t, err)

	want := []string{"job0", "job3"}
	assert.Len(t, got.Jobs, 2)
	assert.Equal(t, want[0], got.Jobs[0].Job.Id)
	assert.Equal(t, want[1], got.Jobs[1].Job.Id)
}

func TestGetInternalJob(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	err := st.CreateJob(&store.Job{
		JobID:     "job0",
		TenantID:  "t0",
		State:     store.JobStateRunning,
		ProjectID: defaultProjectID,
	})
	assert.NoError(t, err)

	srv := NewWorkerServiceServer(st)
	req := &v1.GetInternalJobRequest{Id: "job0", TenantId: "t0"}
	resp, err := srv.GetInternalJob(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, store.JobStateRunning, store.JobState(resp.Job.Status))
}

func TestUpdateJobPhase(t *testing.T) {
	var tests = []struct {
		name      string
		prevState store.JobState
		req       *v1.UpdateJobPhaseRequest
		wantError bool
		wantState store.JobState
	}{
		{
			name:      "no phase",
			req:       &v1.UpdateJobPhaseRequest{},
			wantError: true,
		},
		{
			name:      "phase pre-processed",
			prevState: store.JobStateQueued,
			req: &v1.UpdateJobPhaseRequest{
				Phase:   v1.JobPhase_JOB_PHASE_PREPROCESSED,
				ModelId: "model0",
			},
			wantState: store.JobStateQueued,
		},
		{
			name:      "phase pre-processed, previous state is not queued",
			prevState: store.JobStateRunning,
			req: &v1.UpdateJobPhaseRequest{
				Phase:   v1.JobPhase_JOB_PHASE_PREPROCESSED,
				ModelId: "model0",
			},
			wantError: true,
		},
		{
			name:      "phase job created",
			prevState: store.JobStateQueued,
			req: &v1.UpdateJobPhaseRequest{
				Phase: v1.JobPhase_JOB_PHASE_JOB_CREATED,
			},
			wantState: store.JobStateRunning,
		},
		{
			name:      "phase job created, previous state is not queued",
			prevState: store.JobStatusFailed,
			req: &v1.UpdateJobPhaseRequest{
				Phase: v1.JobPhase_JOB_PHASE_JOB_CREATED,
			},
			wantError: true,
		},
		{
			name:      "phase fine-tuned",
			prevState: store.JobStateRunning,
			req: &v1.UpdateJobPhaseRequest{
				Phase:   v1.JobPhase_JOB_PHASE_FINETUNED,
				ModelId: "model0",
			},
			wantState: store.JobStateSucceeded,
		},
		{
			name:      "phase fine-tuned, previous state is not running",
			prevState: store.JobStateCancelled,
			req: &v1.UpdateJobPhaseRequest{
				Phase:   v1.JobPhase_JOB_PHASE_FINETUNED,
				ModelId: "model0",
			},
			wantError: true,
		},
		{
			name:      "phase job failed",
			prevState: store.JobStateRunning,
			req: &v1.UpdateJobPhaseRequest{
				Phase:   v1.JobPhase_JOB_PHASE_FAILED,
				Message: "error",
			},
			wantState: store.JobStatusFailed,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			st, tearDown := store.NewTest(t)
			defer tearDown()

			const jobID = "job0"
			const tenantID = "t0"
			err := st.CreateJob(&store.Job{
				JobID:    jobID,
				TenantID: tenantID,
				State:    test.prevState,
			})
			assert.NoError(t, err)

			test.req.Id = jobID
			test.req.TenantId = tenantID

			srv := NewWorkerServiceServer(st)
			_, err = srv.UpdateJobPhase(context.Background(), test.req)
			if test.wantError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			job, err := st.GetJobByJobID(jobID)
			assert.NoError(t, err)
			assert.Equal(t, test.wantState, job.State)
		})
	}
}

type noopFileGetClient struct {
	ids map[string]bool
}

func (c *noopFileGetClient) GetFile(ctx context.Context, in *fv1.GetFileRequest, opts ...grpc.CallOption) (*fv1.File, error) {
	if _, ok := c.ids[in.Id]; !ok {
		return nil, status.Error(codes.NotFound, "file not found")
	}

	return &fv1.File{}, nil
}

type noopModelClient struct {
	id string
}

func (c *noopModelClient) GetModel(ctx context.Context, in *mv1.GetModelRequest, opts ...grpc.CallOption) (*mv1.Model, error) {
	if in.Id != c.id {
		return nil, status.Error(codes.NotFound, "model not found")
	}

	return &mv1.Model{}, nil
}

type noopK8sClient struct{}

func (c *noopK8sClient) CancelJob(ctx context.Context, job *v1.Job, namespace string) error {
	return nil
}

func (c *noopK8sClient) CreateSecret(ctx context.Context, name, namespace string, data map[string][]byte) error {
	return nil
}
