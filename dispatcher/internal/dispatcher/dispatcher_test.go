package dispatcher

import (
	"context"
	"testing"
	"time"

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
			State:    store.JobStateCompleted,
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

	pc := &noopPodCreator{}
	d := New(st, pc, time.Second)
	err := d.processPendingJobs(context.Background())
	assert.NoError(t, err)

	wants := map[string]store.JobState{
		jobs[0].JobID: store.JobStateRunning,
		jobs[1].JobID: store.JobStateCompleted,
		jobs[2].JobID: store.JobStateRunning,
	}
	for jobID, want := range wants {
		got, err := st.GetJobByJobID(jobID)
		assert.NoError(t, err)
		assert.Equal(t, want, got.State)
	}
	const wantCounter = 2
	assert.Equal(t, wantCounter, pc.counter)
}

type noopPodCreator struct {
	counter int
}

func (n *noopPodCreator) createPod(ctx context.Context, job *store.Job) error {
	n.counter++
	return nil
}
