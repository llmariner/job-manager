package server

import (
	"context"
	"fmt"
	"testing"

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

			srv := New(st, nil, nil, &noopK8sClientFactory{}, nil, map[string]string{"t0": "img0"})
			resp, err := srv.CreateBatchJob(context.Background(), tc.req)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			_, err = st.GetBatchJobByIDAndProjectID(resp.Id, defaultProjectID)
			assert.NoError(t, err)
		})
	}
}

func TestListBatchJobs(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	for i := 0; i < 10; i++ {
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

	srv := New(st, nil, nil, nil, nil, nil)
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

	srv := New(st, nil, nil, nil, nil, nil)
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
			name:   "transit queued to cancelping",
			state:  store.BatchJobStateQueued,
			action: store.BatchJobQueuedActionCreate,
			want:   &v1.BatchJob{Status: string(store.BatchJobQueuedActionCancel)},
		},
		{
			name:  "transit running to cancelping",
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

			srv := New(st, nil, nil, nil, nil, nil)
			resp, err := srv.CancelBatchJob(context.Background(), &v1.CancelBatchJobRequest{Id: nbID})
			assert.NoError(t, err)
			assert.Equal(t, tc.want.Status, resp.Status)
		})
	}
}
