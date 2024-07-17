package dispatcher

import (
	"context"
	"testing"
	"time"

	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestProcessQueuedJobs(t *testing.T) {
	jobs := []*v1.InternalJob{
		{
			Job: &v1.Job{
				Id: "job0",
			},
			State: v1.InternalJob_QUEUED,
		},
		{
			Job: &v1.Job{
				Id: "job1",
			},
			State: v1.InternalJob_QUEUED,
		},
	}

	ft := &fakeFineTuningWorkerServiceClient{
		jobs:          jobs,
		updatedPhases: map[string]v1.UpdateJobPhaseRequest_Phase{},
	}
	ws := &fakeWorkspaceWorkerServiceClient{}
	jc := &noopJobCreator{}
	pp := &NoopPreProcessor{}
	nc := &noopNotebookManager{}
	d := New(ft, ws, jc, pp, nc, time.Second)
	err := d.processQueuedJobs(context.Background())
	assert.NoError(t, err)

	wants := map[string]v1.UpdateJobPhaseRequest_Phase{
		jobs[0].Job.Id: v1.UpdateJobPhaseRequest_JOB_CREATED,
		jobs[1].Job.Id: v1.UpdateJobPhaseRequest_JOB_CREATED,
	}
	for jobID, want := range wants {
		got, ok := ft.updatedPhases[jobID]
		assert.True(t, ok)
		assert.Equal(t, want, got)
	}
	const wantCounter = 2
	assert.Equal(t, wantCounter, jc.counter)
}

func TestProcessQueuedNotebooks(t *testing.T) {
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

	ft := &fakeFineTuningWorkerServiceClient{}
	ws := &fakeWorkspaceWorkerServiceClient{notebooks: nbs, updatedState: map[string]v1.NotebookState{}}
	jc := &noopJobCreator{}
	pp := &NoopPreProcessor{}
	nc := &noopNotebookManager{}
	d := New(ft, ws, jc, pp, nc, time.Second)
	err := d.processNotebooks(context.Background())
	assert.NoError(t, err)

	wants := map[string]v1.NotebookState{
		nbs[0].Notebook.Id: v1.NotebookState_INITIALIZING,
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

func (n *noopJobCreator) createJob(ctx context.Context, job *v1.InternalJob, presult *PreProcessResult) error {
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

type fakeFineTuningWorkerServiceClient struct {
	jobs          []*v1.InternalJob
	updatedPhases map[string]v1.UpdateJobPhaseRequest_Phase
}

func (c *fakeFineTuningWorkerServiceClient) ListQueuedInternalJobs(ctx context.Context, in *v1.ListQueuedInternalJobsRequest, opts ...grpc.CallOption) (*v1.ListQueuedInternalJobsResponse, error) {
	return &v1.ListQueuedInternalJobsResponse{Jobs: c.jobs}, nil
}

func (c *fakeFineTuningWorkerServiceClient) GetInternalJob(ctx context.Context, in *v1.GetInternalJobRequest, opts ...grpc.CallOption) (*v1.InternalJob, error) {
	for _, job := range c.jobs {
		if job.Job.Id == in.Id {
			return job, nil
		}
	}
	return nil, status.Error(codes.NotFound, "job not found")
}

func (c *fakeFineTuningWorkerServiceClient) UpdateJobPhase(ctx context.Context, in *v1.UpdateJobPhaseRequest, opts ...grpc.CallOption) (*v1.UpdateJobPhaseResponse, error) {
	c.updatedPhases[in.Id] = in.Phase
	return &v1.UpdateJobPhaseResponse{}, nil
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
