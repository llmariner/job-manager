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
			State:        v1.InternalJob_QUEUED,
			QueuedAction: v1.InternalJob_CREATING,
		},
		{
			Job: &v1.Job{
				Id: "job1",
			},
			State:        v1.InternalJob_QUEUED,
			QueuedAction: v1.InternalJob_CANCELING,
		},
	}

	jc := &noopJobCreator{}
	ft := &fakeFineTuningWorkerServiceClient{
		jobs:          jobs,
		updatedPhases: map[string]v1.UpdateJobPhaseRequest_Phase{},
	}
	d := newTestDispatcher()
	d.ftClient = ft
	d.jobManager = jc
	err := d.processQueuedJobs(context.Background())
	assert.NoError(t, err)

	wants := map[string]v1.UpdateJobPhaseRequest_Phase{
		jobs[0].Job.Id: v1.UpdateJobPhaseRequest_JOB_CREATED,
		jobs[1].Job.Id: v1.UpdateJobPhaseRequest_CANCELED,
	}
	for jobID, want := range wants {
		got, ok := ft.updatedPhases[jobID]
		assert.True(t, ok)
		assert.Equal(t, want, got)
	}
	assert.Equal(t, 1, jc.createCounter)
	assert.Equal(t, 1, jc.cancelCounter)
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

	ws := &fakeWorkspaceWorkerServiceClient{notebooks: nbs, updatedState: map[string]v1.NotebookState{}}
	d := newTestDispatcher()
	d.wsClient = ws
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

func TestProcessQueuedBatchJobs(t *testing.T) {
	jobs := []*v1.InternalBatchJob{
		{
			Job: &v1.BatchJob{
				Id: "job0",
			},
			State:        v1.InternalBatchJob_QUEUED,
			QueuedAction: v1.InternalBatchJob_CREATING,
		},
		{
			Job: &v1.BatchJob{
				Id: "job1",
			},
			State:        v1.InternalBatchJob_QUEUED,
			QueuedAction: v1.InternalBatchJob_CANCELING,
		},
		{
			Job: &v1.BatchJob{
				Id: "job2",
			},
			State:        v1.InternalBatchJob_QUEUED,
			QueuedAction: v1.InternalBatchJob_DELETING,
		},
	}

	ws := &fakeBatchWorkerServiceClient{
		jobs:         jobs,
		updatedState: map[string]v1.InternalBatchJob_State{},
	}
	d := newTestDispatcher()
	d.bwClient = ws
	err := d.processBatchJobs(context.Background())
	assert.NoError(t, err)

	wants := map[string]v1.InternalBatchJob_State{
		jobs[0].Job.Id: v1.InternalBatchJob_RUNNING,
		jobs[1].Job.Id: v1.InternalBatchJob_CANCELED,
		jobs[2].Job.Id: v1.InternalBatchJob_DELETED,
	}
	for nbID, want := range wants {
		got, ok := ws.updatedState[nbID]
		assert.True(t, ok)
		assert.Equal(t, want, got)
	}
}

func newTestDispatcher() *D {
	return New(
		&fakeFineTuningWorkerServiceClient{},
		&fakeWorkspaceWorkerServiceClient{},
		&fakeBatchWorkerServiceClient{},
		&noopJobCreator{},
		&NoopPreProcessor{},
		&noopNotebookManager{},
		&noopBatchJobManager{},
		time.Second)
}

type noopJobCreator struct {
	createCounter int
	cancelCounter int
}

func (n *noopJobCreator) createJob(ctx context.Context, job *v1.InternalJob, presult *PreProcessResult) error {
	n.createCounter++
	return nil
}

func (n *noopJobCreator) cancelJob(ctx context.Context, job *v1.InternalJob) error {
	n.cancelCounter++
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

type noopBatchJobManager struct {
}

func (n *noopBatchJobManager) createBatchJob(ctx context.Context, job *v1.InternalBatchJob) error {
	return nil
}

func (n *noopBatchJobManager) cancelBatchJob(ctx context.Context, job *v1.InternalBatchJob) error {
	return nil
}

func (n *noopBatchJobManager) deleteBatchJob(ctx context.Context, job *v1.InternalBatchJob) error {
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

type fakeBatchWorkerServiceClient struct {
	jobs         []*v1.InternalBatchJob
	updatedState map[string]v1.InternalBatchJob_State
}

func (c *fakeBatchWorkerServiceClient) ListQueuedInternalBatchJobs(ctx context.Context, in *v1.ListQueuedInternalBatchJobsRequest, opts ...grpc.CallOption) (*v1.ListQueuedInternalBatchJobsResponse, error) {
	return &v1.ListQueuedInternalBatchJobsResponse{
		Jobs: c.jobs,
	}, nil
}

func (c *fakeBatchWorkerServiceClient) GetInternalBatchJob(ctx context.Context, in *v1.GetInternalBatchJobRequest, opts ...grpc.CallOption) (*v1.InternalBatchJob, error) {
	for _, job := range c.jobs {
		if job.Job.Id == in.Id {
			return job, nil
		}
	}
	return nil, status.Error(codes.NotFound, "batch job not found")
}

func (c *fakeBatchWorkerServiceClient) UpdateBatchJobState(ctx context.Context, in *v1.UpdateBatchJobStateRequest, opts ...grpc.CallOption) (*v1.UpdateBatchJobStateResponse, error) {
	c.updatedState[in.Id] = in.State
	return &v1.UpdateBatchJobStateResponse{}, nil
}
