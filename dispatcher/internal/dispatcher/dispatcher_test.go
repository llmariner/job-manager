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
	nc := &noopNotebookManager{}
	d := New(st, jc, pp, nc, time.Second)
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

func TestProcessQueuedNotebooks(t *testing.T) {
	st, teardown := store.NewTest(t)
	defer teardown()

	nbs := []*store.Notebook{
		{
			NotebookID:   "nb0",
			State:        store.NotebookStateQueued,
			QueuedAction: store.NotebookQueuedActionStart,
			TenantID:     "tid0",
			ProjectID:    "p0",
		},
		{
			NotebookID: "nb1",
			State:      store.NotebookStateStopped,
			TenantID:   "tid0",
			ProjectID:  "p0",
		},
		{
			NotebookID:   "nb2",
			State:        store.NotebookStateQueued,
			QueuedAction: store.NotebookQueuedActionStart,
			TenantID:     "tid1",
			ProjectID:    "p0",
		},
		{
			NotebookID:   "nb3",
			State:        store.NotebookStateQueued,
			QueuedAction: store.NotebookQueuedActionStop,
			TenantID:     "tid1",
			ProjectID:    "p0",
		},
		{
			NotebookID:   "nb4",
			State:        store.NotebookStateQueued,
			QueuedAction: store.NotebookQueuedActionDelete,
			TenantID:     "tid1",
			ProjectID:    "p0",
		},
	}
	for _, nb := range nbs {
		err := st.CreateNotebook(nb)
		assert.NoError(t, err)
	}

	jc := &noopJobCreator{}
	pp := &NoopPreProcessor{}
	nc := &noopNotebookManager{}
	d := New(st, jc, pp, nc, time.Second)
	err := d.processNotebooks(context.Background())
	assert.NoError(t, err)

	wants := map[string]store.NotebookState{
		nbs[0].NotebookID: store.NotebookStateRunning,
		nbs[1].NotebookID: store.NotebookStateStopped,
		nbs[2].NotebookID: store.NotebookStateRunning,
		nbs[3].NotebookID: store.NotebookStateStopped,
	}
	for nbID, want := range wants {
		got, err := st.GetNotebookByIDAndProjectID(nbID, "p0")
		assert.NoError(t, err)
		assert.Equal(t, want, got.State)
	}

	assert.Equal(t, 2, nc.createCounter)
	assert.Equal(t, 1, nc.stopCounter)
	assert.Equal(t, 1, nc.deleteCounter)
}

type noopJobCreator struct {
	counter int
}

func (n *noopJobCreator) createJob(ctx context.Context, job *store.Job, presult *PreProcessResult) error {
	n.counter++
	return nil
}

type noopNotebookManager struct {
	createCounter int
	stopCounter   int
	deleteCounter int
}

func (n *noopNotebookManager) createNotebook(ctx context.Context, nb *store.Notebook) error {
	n.createCounter++
	return nil
}

func (n *noopNotebookManager) stopNotebook(ctx context.Context, nb *store.Notebook) error {
	n.stopCounter++
	return nil
}

func (n *noopNotebookManager) deleteNotebook(ctx context.Context, nb *store.Notebook) error {
	n.deleteCounter++
	return nil
}
