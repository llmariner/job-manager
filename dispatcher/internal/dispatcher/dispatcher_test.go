package dispatcher

import (
	"context"
	"testing"

	"github.com/llm-operator/job-manager/common/pkg/store"
	"github.com/stretchr/testify/assert"
)

func TestProcessPendingJobs(t *testing.T) {
	st, teardown := store.NewTest(t)
	defer teardown()

	jobs := []*store.Job{
		{
			JobID:    "job0",
			State:    store.JobStatePending,
			TenantID: "tid0",
		},
		{
			JobID:    "job1",
			State:    store.JobStateRunning,
			TenantID: "tid0",
		},
		{
			JobID:    "job2",
			State:    store.JobStatePending,
			TenantID: "tid1",
		},
	}
	for _, job := range jobs {
		err := st.CreateJob(job)
		assert.NoError(t, err)
	}

	d := New(st, &noopPodCreator{})
	err := d.processPendingJobs(context.Background())
	assert.NoError(t, err)

	wants := map[string]store.JobState{
		jobs[0].JobID: store.JobStateCompleted,
		jobs[1].JobID: store.JobStateRunning,
		jobs[2].JobID: store.JobStateCompleted,
	}
	for jobID, want := range wants {
		got, err := st.GetJobByJobID(jobID)
		assert.NoError(t, err)
		assert.Equal(t, want, got.State)
	}
}

type noopPodCreator struct {
}

func (n *noopPodCreator) createPod(ctx context.Context, job *store.Job) error {
	return nil
}
