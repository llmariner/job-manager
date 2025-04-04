package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr/testr"
	fv1 "github.com/llmariner/file-manager/api/v1"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/server/internal/cache"
	"github.com/llmariner/job-manager/server/internal/k8s"
	"github.com/llmariner/job-manager/server/internal/scheduler"
	"github.com/llmariner/job-manager/server/internal/store"
	mv1 "github.com/llmariner/model-manager/api/v1"
	"github.com/llmariner/rbac-manager/pkg/auth"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
			name: "success with validation resources",
			req: &v1.CreateJobRequest{
				Model:          modelID,
				TrainingFile:   tFileID,
				ValidationFile: vFileID,
				Suffix:         "suffix0",
				Resources: &v1.Job_Resources{
					GpuCount: int32(4),
				},
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
				&fakeScheduler{},
				&fakeCache{},
				nil,
				nil,
				testr.New(t))
			resp, err := srv.CreateJob(fakeAuthInto(context.Background()), tc.req)
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

	srv := New(st, nil, nil, &noopK8sClientFactory{}, &fakeScheduler{}, &fakeCache{}, nil, nil, testr.New(t))
	ctx := fakeAuthInto(context.Background())
	resp, err := srv.ListJobs(ctx, &v1.ListJobsRequest{Limit: 5})
	assert.NoError(t, err)
	assert.True(t, resp.HasMore)
	assert.Len(t, resp.Data, 5)
	want := []string{"job9", "job8", "job7", "job6", "job5"}
	for i, job := range resp.Data {
		assert.Equal(t, want[i], job.Id)
	}

	resp, err = srv.ListJobs(ctx, &v1.ListJobsRequest{After: resp.Data[4].Id, Limit: 2})
	assert.NoError(t, err)
	assert.True(t, resp.HasMore)
	assert.Len(t, resp.Data, 2)
	want = []string{"job4", "job3"}
	for i, job := range resp.Data {
		assert.Equal(t, want[i], job.Id)
	}

	resp, err = srv.ListJobs(ctx, &v1.ListJobsRequest{After: resp.Data[1].Id, Limit: 3})
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

	jobProto := &v1.Job{
		Id: jobID,
		Resources: &v1.Job_Resources{
			GpuCount: 4,
		},
	}
	msg, err := proto.Marshal(jobProto)
	assert.NoError(t, err)
	err = st.CreateJob(&store.Job{
		JobID:        jobID,
		TenantID:     defaultTenantID,
		ProjectID:    defaultProjectID,
		State:        store.JobStateQueued,
		QueuedAction: store.JobQueuedActionCreate,
		Message:      msg,
	})
	assert.NoError(t, err)

	srv := New(st, nil, nil, &noopK8sClientFactory{}, &fakeScheduler{}, &fakeCache{}, nil, nil, testr.New(t))
	resp, err := srv.GetJob(fakeAuthInto(context.Background()), &v1.GetJobRequest{Id: jobID})
	assert.NoError(t, err)
	assert.Equal(t, string(store.JobQueuedActionCreate), resp.Status)
	assert.Equal(t, int32(4), resp.Resources.GpuCount)
}

func TestJobCancel(t *testing.T) {
	const jobID = "job-1"
	var tcs = []struct {
		name   string
		state  store.JobState
		action store.JobQueuedAction
		want   *v1.Job
	}{
		{
			name:   "transit pending to cancelling",
			state:  store.JobStateQueued,
			action: store.JobQueuedActionCreate,
			want:   &v1.Job{Status: string(store.JobQueuedActionCancel)},
		},
		{
			name:  "transit running to canceled",
			state: store.JobStateRunning,
			want:  &v1.Job{Status: string(store.JobQueuedActionCancel)},
		},
		{
			name:  "keep completed state",
			state: store.JobStateSucceeded,
			want:  &v1.Job{Status: string(store.JobStateSucceeded)},
		},
		{
			name:  "keep canceled state",
			state: store.JobStateCanceled,
			want:  &v1.Job{Status: string(store.JobStateCanceled)},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			st, tearDown := store.NewTest(t)
			defer tearDown()

			err := st.CreateJob(&store.Job{
				JobID:        jobID,
				State:        tc.state,
				QueuedAction: tc.action,
				TenantID:     defaultTenantID,
				ProjectID:    defaultProjectID,
			})
			assert.NoError(t, err)

			srv := New(st, nil, nil, &noopK8sClientFactory{}, &fakeScheduler{}, &fakeCache{}, nil, nil, testr.New(t))
			resp, err := srv.CancelJob(fakeAuthInto(context.Background()), &v1.CancelJobRequest{Id: jobID})
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
			State:     store.JobStateQueued,
			TenantID:  defaultTenantID,
			ClusterID: defaultClusterID,
		},
		{
			State:     store.JobStateRunning,
			TenantID:  defaultTenantID,
			ClusterID: defaultClusterID,
		},
		{
			State:     store.JobStateQueued,
			TenantID:  "different-tenant",
			ClusterID: "different-cluster",
		},
		{
			State:     store.JobStateQueued,
			TenantID:  defaultTenantID,
			ClusterID: defaultClusterID,
		},
	}
	for i, job := range jobs {
		jobProto := &v1.Job{
			Id: fmt.Sprintf("job%d", i),
		}
		msg, err := proto.Marshal(jobProto)
		assert.NoError(t, err)
		assert.NoError(t, st.CreateJob(&store.Job{
			JobID:     jobProto.Id,
			State:     job.State,
			Message:   msg,
			TenantID:  job.TenantID,
			ClusterID: job.ClusterID,
		}))
	}

	srv := NewWorkerServiceServer(st, cache.NewStore(st, testr.New(t)), testr.New(t))
	req := &v1.ListQueuedInternalJobsRequest{}
	got, err := srv.ListQueuedInternalJobs(fakeAuthInto(context.Background()), req)
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
		TenantID:  defaultTenantID,
		State:     store.JobStateRunning,
		ProjectID: defaultProjectID,
	})
	assert.NoError(t, err)

	srv := NewWorkerServiceServer(st, cache.NewStore(st, testr.New(t)), testr.New(t))
	req := &v1.GetInternalJobRequest{Id: "job0"}
	resp, err := srv.GetInternalJob(fakeAuthInto(context.Background()), req)
	assert.NoError(t, err)
	assert.Equal(t, store.JobStateRunning, store.JobState(resp.Job.Status))
}

func TestUpdateJobPhase(t *testing.T) {
	var tests = []struct {
		name       string
		prevState  store.JobState
		prevAction store.JobQueuedAction
		req        *v1.UpdateJobPhaseRequest
		wantError  bool
		wantState  store.JobState
	}{
		{
			name:      "no phase",
			req:       &v1.UpdateJobPhaseRequest{},
			wantError: true,
		},
		{
			name:       "phase pre-processed",
			prevState:  store.JobStateQueued,
			prevAction: store.JobQueuedActionCreate,
			req: &v1.UpdateJobPhaseRequest{
				Phase:   v1.UpdateJobPhaseRequest_PREPROCESSED,
				ModelId: "model0",
			},
			wantState: store.JobStateQueued,
		},
		{
			name:      "phase pre-processed, previous state is not queued",
			prevState: store.JobStateRunning,
			req: &v1.UpdateJobPhaseRequest{
				Phase:   v1.UpdateJobPhaseRequest_PREPROCESSED,
				ModelId: "model0",
			},
			wantError: true,
		},
		{
			name:       "phase job created",
			prevState:  store.JobStateQueued,
			prevAction: store.JobQueuedActionCreate,
			req: &v1.UpdateJobPhaseRequest{
				Phase: v1.UpdateJobPhaseRequest_JOB_CREATED,
			},
			wantState: store.JobStateRunning,
		},
		{
			name:      "phase job created, previous state is not queued",
			prevState: store.JobStateFailed,
			req: &v1.UpdateJobPhaseRequest{
				Phase: v1.UpdateJobPhaseRequest_JOB_CREATED,
			},
			wantError: true,
		},
		{
			name:      "phase fine-tuned",
			prevState: store.JobStateRunning,
			req: &v1.UpdateJobPhaseRequest{
				Phase:   v1.UpdateJobPhaseRequest_FINETUNED,
				ModelId: "model0",
			},
			wantState: store.JobStateSucceeded,
		},
		{
			name:      "phase fine-tuned, previous state is not running",
			prevState: store.JobStateCanceled,
			req: &v1.UpdateJobPhaseRequest{
				Phase:   v1.UpdateJobPhaseRequest_FINETUNED,
				ModelId: "model0",
			},
			wantError: true,
		},
		{
			name:      "phase job failed",
			prevState: store.JobStateRunning,
			req: &v1.UpdateJobPhaseRequest{
				Phase:   v1.UpdateJobPhaseRequest_FAILED,
				Message: "error",
			},
			wantState: store.JobStateFailed,
		},
		{
			name:      "phase recreate",
			prevState: store.JobStateRunning,
			req: &v1.UpdateJobPhaseRequest{
				Phase:   v1.UpdateJobPhaseRequest_RECREATE,
				ModelId: "model0",
			},
			wantState: store.JobStateQueued,
		},
		{
			name:      "phase recreate, previous state is not running",
			prevState: store.JobStateQueued,
			req: &v1.UpdateJobPhaseRequest{
				Phase:   v1.UpdateJobPhaseRequest_RECREATE,
				ModelId: "model0",
			},
			wantError: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			st, tearDown := store.NewTest(t)
			defer tearDown()

			const jobID = "job0"
			err := st.CreateJob(&store.Job{
				JobID:    jobID,
				TenantID: defaultTenantID,
				State:    test.prevState,
			})
			assert.NoError(t, err)

			test.req.Id = jobID

			srv := NewWorkerServiceServer(st, cache.NewStore(st, testr.New(t)), testr.New(t))
			_, err = srv.UpdateJobPhase(fakeAuthInto(context.Background()), test.req)
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

type noopK8sClientFactory struct{}

func (f *noopK8sClientFactory) NewClient(clusterID string, token string) (k8s.Client, error) {
	return &noopK8sClient{}, nil
}

func (f *noopK8sClientFactory) NewDynamicClient(clusterID, token string) (k8s.DynamicClient, error) {
	return &noopDynClient{}, nil
}

type noopK8sClient struct{}

func (c *noopK8sClient) CreateSecret(ctx context.Context, name, namespace string, data map[string][]byte) error {
	return nil
}

func (c *noopK8sClient) CreateConfigMap(ctx context.Context, name, namespace string, data map[string][]byte) error {
	return nil
}

type noopDynClient struct{}

func (c *noopDynClient) PatchResource(ctx context.Context, name, namespace string, gvr schema.GroupVersionResource, data []byte) (*unstructured.Unstructured, error) {
	return nil, nil
}

func (c *noopDynClient) DeleteResource(ctx context.Context, name, namespace string, gvr schema.GroupVersionResource) error {
	return nil
}

type fakeScheduler struct{}

func (s *fakeScheduler) Schedule(userInfo *auth.UserInfo, clusterID string, gpuCount int) (scheduler.SchedulingResult, error) {
	if len(userInfo.AssignedKubernetesEnvs) == 0 {
		return scheduler.SchedulingResult{}, fmt.Errorf("no kuberentes cluster/namespace")
	}
	kenv := userInfo.AssignedKubernetesEnvs[0]
	return scheduler.SchedulingResult{
		ClusterID: kenv.ClusterID,
		Namespace: kenv.Namespace,
	}, nil
}

type fakeCache struct{}

func (c *fakeCache) AddAssumedPod(tenantID, clusterID, key string, gpuCount int) error { return nil }
