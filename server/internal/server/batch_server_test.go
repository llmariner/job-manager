package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr/testr"
	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/server/internal/store"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestCreateBatchJob(t *testing.T) {
	tcs := []struct {
		name    string
		req     *v1.CreateBatchJobRequest
		wantErr bool
	}{
		{
			name: "success",
			req: &v1.CreateBatchJobRequest{
				Image:   "t0",
				Command: "python train.py",
				Scripts: map[string][]byte{"train.py": []byte("dummy-data")},
			},
			wantErr: false,
		},
		{
			name: "no image",
			req: &v1.CreateBatchJobRequest{
				Image:   "",
				Command: "python train.py",
				Scripts: map[string][]byte{"train.py": []byte("dummy-data")},
			},
			wantErr: true,
		},
		{
			name: "no command",
			req: &v1.CreateBatchJobRequest{
				Image:   "t0",
				Command: "",
				Scripts: map[string][]byte{"train.py": []byte("dummy-data")},
			},
			wantErr: true,
		},
		{
			name: "no script",
			req: &v1.CreateBatchJobRequest{
				Image:   "t0",
				Command: "python train.py",
				Scripts: map[string][]byte{},
			},
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			st, tearDown := store.NewTest(t)
			defer tearDown()

			srv := New(st, nil, nil, &noopK8sClientFactory{}, nil, map[string]string{"t0": "img0"}, testr.New(t))
			resp, err := srv.CreateBatchJob(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			_, err = st.GetActiveBatchJobByIDAndProjectID(resp.Id, defaultProjectID)
			assert.NoError(t, err)
		})
	}
}

func TestListBatchJobs(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	for i := 0; i < 11; i++ {
		nbProto := &v1.BatchJob{
			Id: fmt.Sprintf("nb%d", i),
		}
		msg, err := proto.Marshal(nbProto)
		assert.NoError(t, err)
		nb := &store.BatchJob{
			JobID:     nbProto.Id,
			Message:   msg,
			TenantID:  defaultTenantID,
			ProjectID: defaultProjectID,
		}
		err = st.CreateBatchJob(nb)
		assert.NoError(t, err)
	}
	err := st.SetBatchJobState("nb10", 0, store.BatchJobStateDeleted)
	assert.NoError(t, err)

	srv := New(st, nil, nil, nil, nil, nil, testr.New(t))
	resp, err := srv.ListBatchJobs(context.Background(), &v1.ListBatchJobsRequest{Limit: 5})
	assert.NoError(t, err)
	assert.True(t, resp.HasMore)
	assert.Len(t, resp.Jobs, 5)
	want := []string{"nb9", "nb8", "nb7", "nb6", "nb5"}
	for i, batchJob := range resp.Jobs {
		assert.Equal(t, want[i], batchJob.Id)
	}

	resp, err = srv.ListBatchJobs(context.Background(), &v1.ListBatchJobsRequest{After: resp.Jobs[4].Id, Limit: 2})
	assert.NoError(t, err)
	assert.True(t, resp.HasMore)
	assert.Len(t, resp.Jobs, 2)
	want = []string{"nb4", "nb3"}
	for i, batchJob := range resp.Jobs {
		assert.Equal(t, want[i], batchJob.Id)
	}

	resp, err = srv.ListBatchJobs(context.Background(), &v1.ListBatchJobsRequest{After: resp.Jobs[1].Id, Limit: 3})
	assert.NoError(t, err)
	assert.False(t, resp.HasMore)
	assert.Len(t, resp.Jobs, 3)
	want = []string{"nb2", "nb1", "nb0"}
	for i, batchJob := range resp.Jobs {
		assert.Equal(t, want[i], batchJob.Id)
	}
}

func TestGetBatchJob(t *testing.T) {
	const nbID = "n0"

	st, tearDown := store.NewTest(t)
	defer tearDown()

	err := st.CreateBatchJob(&store.BatchJob{
		JobID:        nbID,
		TenantID:     defaultTenantID,
		ProjectID:    defaultProjectID,
		State:        store.BatchJobStateQueued,
		QueuedAction: store.BatchJobQueuedActionCreate,
	})
	assert.NoError(t, err)

	srv := New(st, nil, nil, nil, nil, nil, testr.New(t))
	resp, err := srv.GetBatchJob(context.Background(), &v1.GetBatchJobRequest{Id: nbID})
	assert.NoError(t, err)
	assert.EqualValues(t, store.BatchJobQueuedActionCreate, store.BatchJobState(resp.Status))
}

func TestCancelBatchJob(t *testing.T) {
	const nbID = "nb0"
	var tcs = []struct {
		name   string
		state  store.BatchJobState
		action store.BatchJobQueuedAction
		want   *v1.BatchJob
	}{
		{
			name:   "transit queued to canceling",
			state:  store.BatchJobStateQueued,
			action: store.BatchJobQueuedActionCreate,
			want:   &v1.BatchJob{Status: string(store.BatchJobQueuedActionCancel)},
		},
		{
			name:  "transit running to canceling",
			state: store.BatchJobStateRunning,
			want:  &v1.BatchJob{Status: string(store.BatchJobQueuedActionCancel)},
		},
		{
			name:  "keep failed state",
			state: store.BatchJobStateFailed,
			want:  &v1.BatchJob{Status: string(store.BatchJobStateFailed)},
		},
		{
			name:  "keep cancelped state",
			state: store.BatchJobStateCanceled,
			want:  &v1.BatchJob{Status: string(store.BatchJobStateCanceled)},
		},
		{
			name:  "keep succeeded state",
			state: store.BatchJobStateSucceeded,
			want:  &v1.BatchJob{Status: string(store.BatchJobStateSucceeded)},
		},
		{
			name:   "keep canceling state",
			state:  store.BatchJobStateQueued,
			action: store.BatchJobQueuedActionCancel,
			want:   &v1.BatchJob{Status: string(store.BatchJobQueuedActionCancel)},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			st, tearDown := store.NewTest(t)
			defer tearDown()

			err := st.CreateBatchJob(&store.BatchJob{
				JobID:        nbID,
				State:        tc.state,
				QueuedAction: tc.action,
				TenantID:     defaultTenantID,
				ProjectID:    defaultProjectID,
			})
			assert.NoError(t, err)

			srv := New(st, nil, nil, nil, nil, nil, testr.New(t))
			resp, err := srv.CancelBatchJob(context.Background(), &v1.CancelBatchJobRequest{Id: nbID})
			assert.NoError(t, err)
			assert.Equal(t, tc.want.Status, resp.Status)
		})
	}
}

func TestListQueuedInternalBatchJobs(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	jobs := []*store.BatchJob{
		{
			State:    store.BatchJobStateQueued,
			TenantID: defaultTenantID,
		},
		{
			State:    store.BatchJobStateRunning,
			TenantID: defaultTenantID,
		},
		{
			State:    store.BatchJobStateQueued,
			TenantID: "different-tenant",
		},
		{
			State:    store.BatchJobStateQueued,
			TenantID: defaultTenantID,
		},
		{
			State:    store.BatchJobStateDeleted,
			TenantID: defaultTenantID,
		},
	}
	for i, job := range jobs {
		jobProto := &v1.Job{
			Id: fmt.Sprintf("job%d", i),
		}
		msg, err := proto.Marshal(jobProto)
		assert.NoError(t, err)
		assert.NoError(t, st.CreateBatchJob(&store.BatchJob{
			JobID:    jobProto.Id,
			State:    job.State,
			Message:  msg,
			TenantID: job.TenantID,
		}))
	}

	srv := NewWorkerServiceServer(st, testr.New(t))
	req := &v1.ListQueuedInternalBatchJobsRequest{}
	got, err := srv.ListQueuedInternalBatchJobs(context.Background(), req)
	assert.NoError(t, err)

	want := []string{"job0", "job3"}
	assert.Len(t, got.Jobs, 2)
	assert.Equal(t, want[0], got.Jobs[0].Job.Id)
	assert.Equal(t, want[1], got.Jobs[1].Job.Id)
}

func TestGetInternalBatchJob(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	err := st.CreateBatchJob(&store.BatchJob{
		JobID:     "job0",
		TenantID:  defaultTenantID,
		State:     store.BatchJobStateRunning,
		ProjectID: defaultProjectID,
	})
	assert.NoError(t, err)

	srv := NewWorkerServiceServer(st, testr.New(t))
	req := &v1.GetInternalBatchJobRequest{Id: "job0"}
	resp, err := srv.GetInternalBatchJob(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, store.JobStateRunning, store.JobState(resp.Job.Status))
}

func TestDeleteBatchJob(t *testing.T) {
	const nbID = "nb0"

	st, tearDown := store.NewTest(t)
	defer tearDown()

	err := st.CreateBatchJob(&store.BatchJob{
		JobID:        nbID,
		State:        store.BatchJobStateQueued,
		QueuedAction: store.BatchJobQueuedActionCreate,
		TenantID:     defaultTenantID,
		ProjectID:    defaultProjectID,
	})
	assert.NoError(t, err)

	srv := New(st, nil, nil, nil, nil, nil, testr.New(t))
	_, err = srv.DeleteBatchJob(context.Background(), &v1.DeleteBatchJobRequest{Id: nbID})
	assert.NoError(t, err)
}

func TestUpdateBatchJobState(t *testing.T) {
	var tests = []struct {
		name       string
		prevState  store.BatchJobState
		prevAction store.BatchJobQueuedAction
		state      v1.InternalBatchJob_State
		wantError  bool
		wantState  store.BatchJobState
	}{
		{
			name:      "no state",
			wantError: true,
		},
		{
			name:       "unknown state",
			prevState:  store.BatchJobStateQueued,
			prevAction: store.BatchJobQueuedActionCreate,
			state:      9999,
			wantError:  true,
		},
		{
			name:      "same state",
			prevState: store.BatchJobStateRunning,
			state:     v1.InternalBatchJob_RUNNING,
			wantState: store.BatchJobStateRunning,
		},
		{
			name:       "set running state",
			prevState:  store.BatchJobStateQueued,
			prevAction: store.BatchJobQueuedActionCreate,
			state:      v1.InternalBatchJob_RUNNING,
			wantState:  store.BatchJobStateRunning,
		},
		{
			name:      "set running state, previous state is not queued",
			prevState: store.BatchJobStateSucceeded,
			state:     v1.InternalBatchJob_RUNNING,
			wantError: true,
		},
		{
			name:       "set cancel state",
			prevState:  store.BatchJobStateQueued,
			prevAction: store.BatchJobQueuedActionCancel,
			state:      v1.InternalBatchJob_CANCELED,
			wantState:  store.BatchJobStateCanceled,
		},
		{
			name:      "set cancel state, previous state is not queued",
			prevState: store.BatchJobStateSucceeded,
			state:     v1.InternalBatchJob_CANCELED,
			wantError: true,
		},
		{
			name:       "set delete state",
			prevState:  store.BatchJobStateQueued,
			prevAction: store.BatchJobQueuedActionDelete,
			state:      v1.InternalBatchJob_DELETED,
			wantState:  store.BatchJobStateDeleted,
		},
		{
			name:      "set delete state, previous state is not queued",
			prevState: store.BatchJobStateSucceeded,
			state:     v1.InternalBatchJob_DELETED,
			wantError: true,
		},
		{
			name:      "set failed state",
			prevState: store.BatchJobStateRunning,
			state:     v1.InternalBatchJob_FAILED,
			wantState: store.BatchJobStateFailed,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			st, tearDown := store.NewTest(t)
			defer tearDown()

			const batchJobID = "batchJob0"
			err := st.CreateBatchJob(&store.BatchJob{
				JobID:        batchJobID,
				TenantID:     defaultTenantID,
				State:        test.prevState,
				QueuedAction: test.prevAction,
			})
			assert.NoError(t, err)

			srv := NewWorkerServiceServer(st, testr.New(t))
			_, err = srv.UpdateBatchJobState(context.Background(), &v1.UpdateBatchJobStateRequest{
				Id:    batchJobID,
				State: test.state,
			})
			if test.wantError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			batchJob, err := st.GetBatchJobByID(batchJobID)
			assert.NoError(t, err)
			assert.Equal(t, test.wantState, batchJob.State)
		})
	}
}
