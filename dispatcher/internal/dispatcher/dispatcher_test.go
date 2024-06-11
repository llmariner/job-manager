package dispatcher

import (
	"context"
	"testing"
	"time"

	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
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

	ws := &fakeWorkspaceWorkerServiceClient{}
	jc := &noopJobCreator{}
	pp := &NoopPreProcessor{}
	nc := &noopNotebookManager{}
	d := New(st, ws, jc, pp, nc, time.Second)
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

	nbs := []*v1.InternalNotebook{
		{
			Notebook: &v1.Notebook{
				Id: "nb0",
			},
			State:        v1.NotebookState_QUEUED,
			QueuedAction: v1.NotebookQueuedAction_STARTING,
		},
		{
			Notebook: &v1.Notebook{
				Id: "nb1",
			},
			State:        v1.NotebookState_QUEUED,
			QueuedAction: v1.NotebookQueuedAction_STOPPING,
		},
		{
			Notebook: &v1.Notebook{
				Id: "nb2",
			},
			State:        v1.NotebookState_QUEUED,
			QueuedAction: v1.NotebookQueuedAction_DELETING,
		},
	}

	ws := &fakeWorkspaceWorkerServiceClient{notebooks: nbs, updatedState: map[string]v1.NotebookState{}}
	jc := &noopJobCreator{}
	pp := &NoopPreProcessor{}
	nc := &noopNotebookManager{}
	d := New(st, ws, jc, pp, nc, time.Second)
	err := d.processNotebooks(context.Background())
	assert.NoError(t, err)

	wants := map[string]v1.NotebookState{
		nbs[0].Notebook.Id: v1.NotebookState_RUNNING,
		nbs[1].Notebook.Id: v1.NotebookState_STOPPED,
		nbs[2].Notebook.Id: v1.NotebookState_DELETED,
	}
	for nbID, want := range wants {
		got, ok := ws.updatedState[nbID]
		assert.True(t, ok)
		assert.Equal(t, want, got)
	}
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

func (n *noopNotebookManager) createNotebook(ctx context.Context, nb *v1.InternalNotebook) error {
	n.createCounter++
	return nil
}

func (n *noopNotebookManager) stopNotebook(ctx context.Context, nb *v1.InternalNotebook) error {
	n.stopCounter++
	return nil
}

func (n *noopNotebookManager) deleteNotebook(ctx context.Context, nb *v1.InternalNotebook) error {
	n.deleteCounter++
	return nil
}

type fakeWorkspaceWorkerServiceClient struct {
	notebooks    []*v1.InternalNotebook
	updatedState map[string]v1.NotebookState
}

func (c *fakeWorkspaceWorkerServiceClient) ListQueuedInternalNotebooks(ctx context.Context, in *v1.ListQueuedInternalNotebooksRequest, opts ...grpc.CallOption) (*v1.ListQueuedInternalNotebooksResponse, error) {
	return &v1.ListQueuedInternalNotebooksResponse{
		Notebooks: c.notebooks,
	}, nil
}

func (c *fakeWorkspaceWorkerServiceClient) UpdateNotebookState(ctx context.Context, in *v1.UpdateNotebookStateRequest, opts ...grpc.CallOption) (*v1.UpdateNotebookStateResponse, error) {
	c.updatedState[in.Id] = in.State
	return &v1.UpdateNotebookStateResponse{}, nil
}
