package server

import (
	"context"
	"fmt"
	"testing"

	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
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

			srv := New(st, nil, nil, nil, map[string]string{"t0": "img0"})
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("Authorization", "dummy"))
			resp, err := srv.CreateNotebook(ctx, tc.req)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			_, err = st.GetNotebookByIDAndProjectID(resp.Id, defaultProjectID)
			assert.NoError(t, err)
		})
	}
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
			TenantID:   fakeTenantID,
			ProjectID:  defaultProjectID,
		}
		err = st.CreateNotebook(nb)
		assert.NoError(t, err)
	}

	srv := New(st, nil, nil, nil, nil)
	resp, err := srv.ListNotebooks(context.Background(), &v1.ListNotebooksRequest{Limit: 5})
	assert.NoError(t, err)
	assert.True(t, resp.HasMore)
	assert.Len(t, resp.Notebooks, 5)
	want := []string{"nb9", "nb8", "nb7", "nb6", "nb5"}
	for i, notebook := range resp.Notebooks {
		assert.Equal(t, want[i], notebook.Id)
	}

	resp, err = srv.ListNotebooks(context.Background(), &v1.ListNotebooksRequest{After: resp.Notebooks[4].Id, Limit: 2})
	assert.NoError(t, err)
	assert.True(t, resp.HasMore)
	assert.Len(t, resp.Notebooks, 2)
	want = []string{"nb4", "nb3"}
	for i, notebook := range resp.Notebooks {
		assert.Equal(t, want[i], notebook.Id)
	}

	resp, err = srv.ListNotebooks(context.Background(), &v1.ListNotebooksRequest{After: resp.Notebooks[1].Id, Limit: 3})
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
		NotebookID: nbID,
		TenantID:   fakeTenantID,
		ProjectID:  defaultProjectID,
		State:      store.NotebookStateQueued,
	})
	assert.NoError(t, err)

	srv := New(st, nil, nil, nil, nil)
	resp, err := srv.GetNotebook(context.Background(), &v1.GetNotebookRequest{Id: nbID})
	assert.NoError(t, err)
	assert.Equal(t, store.NotebookStateQueued, store.NotebookState(resp.Status))
}

func TestStopNotebook(t *testing.T) {
	const nbID = "nb0"
	var tcs = []struct {
		name  string
		state store.NotebookState
		want  *v1.Notebook
	}{
		{
			name:  "transit queued to stopping",
			state: store.NotebookStateQueued,
			want:  &v1.Notebook{Status: string(store.NotebookStateStopping)},
		},
		{
			name:  "transit running to stopping",
			state: store.NotebookStateRunning,
			want:  &v1.Notebook{Status: string(store.NotebookStateStopping)},
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
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			st, tearDown := store.NewTest(t)
			defer tearDown()

			err := st.CreateNotebook(&store.Notebook{NotebookID: nbID, State: tc.state, TenantID: fakeTenantID, ProjectID: defaultProjectID})
			assert.NoError(t, err)

			srv := New(st, nil, nil, nil, nil)
			resp, err := srv.StopNotebook(context.Background(), &v1.StopNotebookRequest{Id: nbID})
			assert.NoError(t, err)
			assert.Equal(t, tc.want.Status, resp.Status)
		})
	}
}

func TestStartNotebook(t *testing.T) {
	const nbID = "nb0"
	var tcs = []struct {
		name  string
		state store.NotebookState
		want  *v1.Notebook
	}{
		{
			name:  "transit stopping to queued",
			state: store.NotebookStateQueued,
			want:  &v1.Notebook{Status: string(store.NotebookStateQueued)},
		},
		{
			name:  "transit stopped to queued",
			state: store.NotebookStateStopped,
			want:  &v1.Notebook{Status: string(store.NotebookStateQueued)},
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
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			st, tearDown := store.NewTest(t)
			defer tearDown()

			err := st.CreateNotebook(&store.Notebook{NotebookID: nbID, State: tc.state, TenantID: fakeTenantID, ProjectID: defaultProjectID})
			assert.NoError(t, err)

			srv := New(st, nil, nil, nil, nil)
			resp, err := srv.StartNotebook(context.Background(), &v1.StartNotebookRequest{Id: nbID})
			assert.NoError(t, err)
			assert.Equal(t, tc.want.Status, resp.Status)
		})
	}
}
