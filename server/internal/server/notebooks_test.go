package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr/testr"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/server/internal/store"
	rbacv1 "github.com/llmariner/rbac-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestCreateNotebook(t *testing.T) {
	tcs := []struct {
		name    string
		req     *v1.CreateNotebookRequest
		wantErr bool
	}{
		{
			name: "success (image type)",
			req: &v1.CreateNotebookRequest{
				Name: "nb0",
				Image: &v1.CreateNotebookRequest_Image{
					Image: &v1.CreateNotebookRequest_Image_Type{Type: "t0"},
				},
			},
			wantErr: false,
		},
		{
			name: "success (image uri)",
			req: &v1.CreateNotebookRequest{
				Name: "nb0",
				Image: &v1.CreateNotebookRequest_Image{
					Image: &v1.CreateNotebookRequest_Image_Uri{Uri: "img0"},
				},
			},
			wantErr: false,
		},
		{
			name: "no image",
			req: &v1.CreateNotebookRequest{
				Name:  "nb0",
				Image: &v1.CreateNotebookRequest_Image{},
			},
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			st, tearDown := store.NewTest(t)
			defer tearDown()

			srv := New(st, nil, nil, &noopK8sClientFactory{}, &fakeScheduler{}, map[string]string{"t0": "img0"}, nil, testr.New(t))
			resp, err := srv.CreateNotebook(fakeAuthInto(context.Background()), tc.req)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			nb, err := st.GetNotebookByIDAndProjectID(resp.Id, defaultProjectID)
			assert.NoError(t, err)
			expProj := &rbacv1.Project{
				Id: defaultProjectID,
				AssignedKubernetesEnvs: []*rbacv1.Project_AssignedKubernetesEnv{
					{
						ClusterId: defaultClusterID,
						Namespace: "default",
					},
				},
			}
			expMsg, err := proto.Marshal(expProj)
			assert.NoError(t, err)
			assert.Equal(t, expMsg, nb.ProjectMessage)
		})
	}
}

func TestCreateNotebook_SameName(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	srv := New(st, nil, nil, &noopK8sClientFactory{}, &fakeScheduler{}, map[string]string{"t0": "img0"}, nil, testr.New(t))
	wsrv := NewWorkerServiceServer(st, testr.New(t))

	req := &v1.CreateNotebookRequest{
		Name: "nb0",
		Image: &v1.CreateNotebookRequest_Image{
			Image: &v1.CreateNotebookRequest_Image_Uri{Uri: "img0"},
		},
	}
	ctx := fakeAuthInto(context.Background())
	resp, err := srv.CreateNotebook(ctx, req)
	assert.NoError(t, err)

	_, err = st.GetNotebookByIDAndProjectID(resp.Id, defaultProjectID)
	assert.NoError(t, err)

	// Create another notebook with the same name.
	_, err = srv.CreateNotebook(ctx, req)
	assert.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))

	// Delete the notebook.
	_, err = srv.DeleteNotebook(ctx, &v1.DeleteNotebookRequest{Id: resp.Id})
	assert.NoError(t, err)

	_, err = wsrv.UpdateNotebookState(ctx, &v1.UpdateNotebookStateRequest{
		Id:    resp.Id,
		State: v1.NotebookState_DELETED,
	})
	assert.NoError(t, err)

	// Can create a notebook with the same name.
	_, err = srv.CreateNotebook(ctx, req)
	assert.NoError(t, err)
}

func TestListNotebooks(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	for i := 0; i < 10; i++ {
		nbProto := &v1.Notebook{
			Id: fmt.Sprintf("nb%d", i),
		}
		msg, err := proto.Marshal(nbProto)
		assert.NoError(t, err)
		nb := &store.Notebook{
			NotebookID: nbProto.Id,
			Message:    msg,
			TenantID:   defaultTenantID,
			ProjectID:  defaultProjectID,
		}
		err = st.CreateNotebook(nb)
		assert.NoError(t, err)
	}

	srv := New(st, nil, nil, nil, nil, nil, nil, testr.New(t))
	ctx := fakeAuthInto(context.Background())
	resp, err := srv.ListNotebooks(ctx, &v1.ListNotebooksRequest{Limit: 5})
	assert.NoError(t, err)
	assert.True(t, resp.HasMore)
	assert.Len(t, resp.Notebooks, 5)
	want := []string{"nb9", "nb8", "nb7", "nb6", "nb5"}
	for i, notebook := range resp.Notebooks {
		assert.Equal(t, want[i], notebook.Id)
	}

	resp, err = srv.ListNotebooks(ctx, &v1.ListNotebooksRequest{After: resp.Notebooks[4].Id, Limit: 2})
	assert.NoError(t, err)
	assert.True(t, resp.HasMore)
	assert.Len(t, resp.Notebooks, 2)
	want = []string{"nb4", "nb3"}
	for i, notebook := range resp.Notebooks {
		assert.Equal(t, want[i], notebook.Id)
	}

	resp, err = srv.ListNotebooks(ctx, &v1.ListNotebooksRequest{After: resp.Notebooks[1].Id, Limit: 3})
	assert.NoError(t, err)
	assert.False(t, resp.HasMore)
	assert.Len(t, resp.Notebooks, 3)
	want = []string{"nb2", "nb1", "nb0"}
	for i, notebook := range resp.Notebooks {
		assert.Equal(t, want[i], notebook.Id)
	}
}

func TestGetNotebook(t *testing.T) {
	const nbID = "n0"

	st, tearDown := store.NewTest(t)
	defer tearDown()

	err := st.CreateNotebook(&store.Notebook{
		NotebookID:   nbID,
		TenantID:     defaultTenantID,
		ProjectID:    defaultProjectID,
		State:        store.NotebookStateQueued,
		QueuedAction: store.NotebookQueuedActionStart,
	})
	assert.NoError(t, err)

	srv := New(st, nil, nil, nil, nil, nil, nil, testr.New(t))
	resp, err := srv.GetNotebook(fakeAuthInto(context.Background()), &v1.GetNotebookRequest{Id: nbID})
	assert.NoError(t, err)
	assert.EqualValues(t, store.NotebookQueuedActionStart, store.NotebookState(resp.Status))
}

func TestStopNotebook(t *testing.T) {
	const nbID = "nb0"
	var tcs = []struct {
		name   string
		state  store.NotebookState
		action store.NotebookQueuedAction
		want   *v1.Notebook
	}{
		{
			name:   "transit queued to stopping",
			state:  store.NotebookStateQueued,
			action: store.NotebookQueuedActionStart,
			want:   &v1.Notebook{Status: string(store.NotebookQueuedActionStop)},
		},
		{
			name:  "transit running to stopping",
			state: store.NotebookStateRunning,
			want:  &v1.Notebook{Status: string(store.NotebookQueuedActionStop)},
		},
		{
			name:  "transit requeued to stopping",
			state: store.NotebookStateRequeued,
			want:  &v1.Notebook{Status: string(store.NotebookQueuedActionStop)},
		},
		{
			name:  "keep failed state",
			state: store.NotebookStateFailed,
			want:  &v1.Notebook{Status: string(store.NotebookStateFailed)},
		},
		{
			name:  "keep stopped state",
			state: store.NotebookStateStopped,
			want:  &v1.Notebook{Status: string(store.NotebookStateStopped)},
		},
		{
			name:   "keep deleting state",
			state:  store.NotebookStateQueued,
			action: store.NotebookQueuedActionDelete,
			want:   &v1.Notebook{Status: string(store.NotebookQueuedActionDelete)},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			st, tearDown := store.NewTest(t)
			defer tearDown()

			err := st.CreateNotebook(&store.Notebook{
				NotebookID:   nbID,
				State:        tc.state,
				QueuedAction: tc.action,
				TenantID:     defaultTenantID,
				ProjectID:    defaultProjectID,
			})
			assert.NoError(t, err)

			srv := New(st, nil, nil, nil, nil, nil, nil, testr.New(t))
			resp, err := srv.StopNotebook(fakeAuthInto(context.Background()), &v1.StopNotebookRequest{Id: nbID})
			assert.NoError(t, err)
			assert.Equal(t, tc.want.Status, resp.Status)
		})
	}
}

func TestStartNotebook(t *testing.T) {
	const nbID = "nb0"
	var tcs = []struct {
		name   string
		state  store.NotebookState
		action store.NotebookQueuedAction
		want   *v1.Notebook
	}{
		{
			name:   "transit stopping to queued",
			state:  store.NotebookStateQueued,
			action: store.NotebookQueuedActionStop,
			want:   &v1.Notebook{Status: string(store.NotebookQueuedActionStart)},
		},
		{
			name:  "transit stopped to queued",
			state: store.NotebookStateStopped,
			want:  &v1.Notebook{Status: string(store.NotebookQueuedActionStart)},
		},
		{
			name:  "keep failed state",
			state: store.NotebookStateFailed,
			want:  &v1.Notebook{Status: string(store.NotebookStateFailed)},
		},
		{
			name:  "keep running state",
			state: store.NotebookStateRunning,
			want:  &v1.Notebook{Status: string(store.NotebookStateRunning)},
		},
		{
			name:   "keep deleting state",
			state:  store.NotebookStateQueued,
			action: store.NotebookQueuedActionDelete,
			want:   &v1.Notebook{Status: string(store.NotebookQueuedActionDelete)},
		},
		{
			name:  "keep requeued state",
			state: store.NotebookStateRequeued,
			want:  &v1.Notebook{Status: string(store.NotebookStateRequeued)},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			st, tearDown := store.NewTest(t)
			defer tearDown()

			err := st.CreateNotebook(&store.Notebook{
				NotebookID:   nbID,
				State:        tc.state,
				QueuedAction: tc.action,
				TenantID:     defaultTenantID,
				ProjectID:    defaultProjectID,
			})
			assert.NoError(t, err)

			srv := New(st, nil, nil, nil, nil, nil, nil, testr.New(t))
			resp, err := srv.StartNotebook(fakeAuthInto(context.Background()), &v1.StartNotebookRequest{Id: nbID})
			assert.NoError(t, err)
			assert.Equal(t, tc.want.Status, resp.Status)
		})
	}
}

func TestDeleteNotebook(t *testing.T) {
	const nbID = "nb0"

	st, tearDown := store.NewTest(t)
	defer tearDown()

	err := st.CreateNotebook(&store.Notebook{
		NotebookID:   nbID,
		State:        store.NotebookStateQueued,
		QueuedAction: store.NotebookQueuedActionStart,
		TenantID:     defaultTenantID,
		ProjectID:    defaultProjectID,
	})
	assert.NoError(t, err)

	srv := New(st, nil, nil, nil, nil, nil, nil, testr.New(t))
	_, err = srv.DeleteNotebook(fakeAuthInto(context.Background()), &v1.DeleteNotebookRequest{Id: nbID})
	assert.NoError(t, err)
}

func TestListQueuedInternalNotebooks(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	notebooks := []*store.Notebook{
		{
			State:     store.NotebookStateQueued,
			TenantID:  defaultTenantID,
			ClusterID: defaultClusterID,
		},
		{
			State:     store.NotebookStateRunning,
			TenantID:  defaultTenantID,
			ClusterID: defaultClusterID,
		},
		{
			State:     store.NotebookStateQueued,
			TenantID:  "different-tenant",
			ClusterID: "different-cluster",
		},
		{
			State:     store.NotebookStateQueued,
			TenantID:  defaultTenantID,
			ClusterID: defaultClusterID,
		},
	}
	for i, notebook := range notebooks {
		notebookProto := &v1.Notebook{
			Id: fmt.Sprintf("nb%d", i),
		}
		msg, err := proto.Marshal(notebookProto)
		assert.NoError(t, err)
		assert.NoError(t, st.CreateNotebook(&store.Notebook{
			NotebookID: notebookProto.Id,
			State:      notebook.State,
			Message:    msg,
			TenantID:   notebook.TenantID,
			ClusterID:  notebook.ClusterID,
		}))
	}

	srv := NewWorkerServiceServer(st, testr.New(t))
	req := &v1.ListQueuedInternalNotebooksRequest{}
	got, err := srv.ListQueuedInternalNotebooks(fakeAuthInto(context.Background()), req)
	assert.NoError(t, err)

	want := []string{"nb0", "nb3"}
	assert.Len(t, got.Notebooks, 2)
	assert.Equal(t, want[0], got.Notebooks[0].Notebook.Id)
	assert.Equal(t, want[1], got.Notebooks[1].Notebook.Id)
}

func TestUpdateNotebookState(t *testing.T) {
	var tests = []struct {
		name       string
		prevState  store.NotebookState
		prevAction store.NotebookQueuedAction
		state      v1.NotebookState
		wantError  bool
		wantState  store.NotebookState
	}{
		{
			name:      "no state",
			wantError: true,
		},
		{
			name:       "unknown state",
			prevState:  store.NotebookStateQueued,
			prevAction: store.NotebookQueuedActionStart,
			state:      9999,
			wantError:  true,
		},
		{
			name:      "same state",
			prevState: store.NotebookStateRunning,
			state:     v1.NotebookState_RUNNING,
			wantState: store.NotebookStateRunning,
		},
		{
			name:       "set initializing state",
			prevState:  store.NotebookStateQueued,
			prevAction: store.NotebookQueuedActionStart,
			state:      v1.NotebookState_INITIALIZING,
			wantState:  store.NotebookStateInitializing,
		},
		{
			name:      "set initializing state, previous state is not queued",
			prevState: store.NotebookStateRunning,
			state:     v1.NotebookState_INITIALIZING,
			wantError: true,
		},
		{
			name:       "set initializing state, previous action is not starting",
			prevState:  store.NotebookStateQueued,
			prevAction: store.NotebookQueuedActionDelete,
			state:      v1.NotebookState_INITIALIZING,
			wantError:  true,
		},
		{
			name:      "set running state",
			prevState: store.NotebookStateInitializing,
			state:     v1.NotebookState_RUNNING,
			wantState: store.NotebookStateRunning,
		},
		{
			name:       "set running state, previous state is queued",
			prevState:  store.NotebookStateQueued,
			prevAction: store.NotebookQueuedActionRequeue,
			state:      v1.NotebookState_RUNNING,
			wantState:  store.NotebookStateQueued,
		},
		{
			name:      "set running state, previous state is not initializing nor queued",
			prevState: store.NotebookStateStopped,
			state:     v1.NotebookState_RUNNING,
			wantError: true,
		},
		{
			name:       "set stopped state",
			prevState:  store.NotebookStateQueued,
			prevAction: store.NotebookQueuedActionStop,
			state:      v1.NotebookState_STOPPED,
			wantState:  store.NotebookStateStopped,
		},
		{
			name:       "set stopped state, previous action is not stopping",
			prevState:  store.NotebookStateQueued,
			prevAction: store.NotebookQueuedActionDelete,
			state:      v1.NotebookState_STOPPED,
			wantError:  true,
		},
		{
			name:       "set deleted state",
			prevState:  store.NotebookStateQueued,
			prevAction: store.NotebookQueuedActionDelete,
			state:      v1.NotebookState_DELETED,
			wantState:  store.NotebookStateDeleted,
		},
		{
			name:       "set deleted state, previous action is not deleting",
			prevState:  store.NotebookStateQueued,
			prevAction: store.NotebookQueuedActionStop,
			state:      v1.NotebookState_DELETED,
			wantError:  true,
		},
		{
			name:       "set requeued state, previous action is requeueing",
			prevState:  store.NotebookStateQueued,
			prevAction: store.NotebookQueuedActionRequeue,
			state:      v1.NotebookState_REQUEUED,
			wantState:  store.NotebookStateRequeued,
		},
		{
			name:       "set requeued state, previous action is requeueing",
			prevState:  store.NotebookStateRequeued,
			prevAction: store.NotebookQueuedActionRequeue,
			state:      v1.NotebookState_DELETED,
			wantError:  true,
		},
		{
			name:       "set stopped state, previous state is requeued",
			prevState:  store.NotebookStateRequeued,
			prevAction: store.NotebookQueuedActionRequeue,
			state:      v1.NotebookState_STOPPED,
			wantError:  true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			st, tearDown := store.NewTest(t)
			defer tearDown()

			const notebookID = "notebook0"
			err := st.CreateNotebook(&store.Notebook{
				NotebookID:   notebookID,
				TenantID:     defaultTenantID,
				State:        test.prevState,
				QueuedAction: test.prevAction,
			})
			assert.NoError(t, err)

			srv := NewWorkerServiceServer(st, testr.New(t))
			_, err = srv.UpdateNotebookState(fakeAuthInto(context.Background()), &v1.UpdateNotebookStateRequest{
				Id:    notebookID,
				State: test.state,
			})
			if test.wantError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			notebook, err := st.GetNotebookByID(notebookID)
			assert.NoError(t, err)
			assert.Equal(t, test.wantState, notebook.State)
		})
	}
}
