package server

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr/testr"
	"github.com/llmariner/job-manager/server/internal/store"
	"github.com/llmariner/rbac-manager/pkg/auth"
	"github.com/stretchr/testify/assert"
)

func TestRescheduleNotebooks(t *testing.T) {
	var tests = []struct {
		name       string
		prevState  store.NotebookState
		prevAction store.NotebookQueuedAction
		wantError  bool
		wantState  store.NotebookState
		wantAction store.NotebookQueuedAction
		waitTime   time.Duration
	}{

		{
			name:       "initializing timeout",
			prevState:  store.NotebookStateInitializing,
			prevAction: store.NotebookQueuedActionStart,
			wantState:  store.NotebookStateQueued,
			wantAction: store.NotebookQueuedActionRequeue,
		},
		{
			name:       "reschedule",
			prevState:  store.NotebookStateRequeued,
			prevAction: store.NotebookQueuedActionRequeue,
			wantState:  store.NotebookStateQueued,
			wantAction: store.NotebookQueuedActionStart,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			st, tearDown := store.NewTest(t)
			defer tearDown()

			const notebookID = "notebook0"
			userInfo := &auth.UserInfo{
				TenantID: defaultTenantID,
				AssignedKubernetesEnvs: []auth.AssignedKubernetesEnv{
					{
						ClusterID: "cluster0",
						Namespace: "namespace0",
					},
				},
			}
			proj, err := toProjectMessage(userInfo)
			assert.NoError(t, err)
			err = st.CreateNotebook(&store.Notebook{
				NotebookID:     notebookID,
				TenantID:       defaultTenantID,
				State:          test.prevState,
				QueuedAction:   test.prevAction,
				ProjectMessage: proj,
			})
			assert.NoError(t, err)
			time.Sleep(time.Second * 2)

			srv := New(st, nil, nil, &noopK8sClientFactory{}, &fakeScheduler{}, map[string]string{"t0": "img0"}, nil, testr.New(t))
			err = srv.rescheduleNotebooks(context.Background(), time.Second)
			assert.NoError(t, err)

			notebook, err := st.GetNotebookByID(notebookID)
			assert.NoError(t, err)
			assert.Equal(t, test.wantState, notebook.State)
			assert.Equal(t, test.wantAction, notebook.QueuedAction)
		})
	}
}
