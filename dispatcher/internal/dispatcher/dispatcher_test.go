package dispatcher

import (
	"context"
	"testing"
	"time"

	"github.com/llm-operator/job-manager/common/pkg/store"
	"github.com/stretchr/testify/assert"
)

func TestProcessQueuedJobs(t *testing.T) {
	st, teardown := store.NewTest(t)
	defer teardown()

	jobs := []*store.Job{
		{
			JobID:    "job0",
			State:    store.JobStateQueued,
			TenantID: "tid0",
		},
		{
			JobID:    "job1",
			State:    store.JobStateSucceeded,
			TenantID: "tid0",
		},
		{
			JobID:    "job2",
			State:    store.JobStateQueued,
			TenantID: "tid1",
		},
	}
	for _, job := range jobs {
		err := st.CreateJob(job)
		assert.NoError(t, err)
	}

	jc := &noopJobCreator{}
	pp := &NoopPreProcessor{}
	d := New(st, jc, pp, time.Second)
	err := d.processQueuedJobs(context.Background())
	assert.NoError(t, err)

	wants := map[string]store.JobState{
		jobs[0].JobID: store.JobStateRunning,
		jobs[1].JobID: store.JobStateSucceeded,
		jobs[2].JobID: store.JobStateRunning,
	}
	for jobID, want := range wants {
		got, err := st.GetJobByJobID(jobID)
		assert.NoError(t, err)
		assert.Equal(t, want, got.State)
	}
	const wantCounter = 2
	assert.Equal(t, wantCounter, jc.counter)
}

type noopJobCreator struct {
	counter int
}

func (n *noopJobCreator) createJob(ctx context.Context, job *store.Job, presult *PreProcessResult) error {
	n.counter++
	return nil
}
