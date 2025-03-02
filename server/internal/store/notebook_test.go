package store

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestCreateAndGetNotebook(t *testing.T) {
	const (
		name      = "notebook0"
		projectID = "pid0"
	)
	st, teardown := NewTest(t)
	defer teardown()

	_, err := st.GetActiveNotebookByNameAndProjectID(name, projectID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))

	nb := &Notebook{
		Name:      name,
		ProjectID: projectID,
		State:     NotebookStateRunning,
	}
	err = st.CreateNotebook(nb)
	assert.NoError(t, err)

	_, err = st.GetActiveNotebookByNameAndProjectID(name, projectID)
	assert.NoError(t, err)

	nb.State = NotebookStateDeleted
	err = st.db.Save(nb).Error
	assert.NoError(t, err)

	_, err = st.GetActiveNotebookByNameAndProjectID(name, projectID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestListQueuedNotebooksByTenantIDAndClusterID(t *testing.T) {
	const (
		projectID = "pid0"
		tenantID  = "tid0"
		clusterID = "cid0"
	)
	st, teardown := NewTest(t)
	defer teardown()

	nbs := []*Notebook{
		&Notebook{
			NotebookID: "nb0",
			Name:       "notebook0",
			TenantID:   tenantID,
			ClusterID:  clusterID,
			ProjectID:  projectID,
			State:      NotebookStateQueued,
		},
		&Notebook{
			NotebookID: "nb1",
			Name:       "notebook1",
			TenantID:   tenantID,
			ClusterID:  clusterID,
			ProjectID:  projectID,
			State:      NotebookStateRequeued,
		},
		&Notebook{
			NotebookID: "nb2",
			Name:       "notebook2",
			TenantID:   tenantID,
			ClusterID:  clusterID,
			ProjectID:  projectID,
			State:      NotebookStateRunning,
		},
	}
	for _, nb := range nbs {
		err := st.CreateNotebook(nb)
		assert.NoError(t, err)
	}

	exp := map[string]*Notebook{
		nbs[0].NotebookID: nbs[0],
	}
	got, err := st.ListQueuedNotebooksByTenantIDAndClusterID(tenantID, clusterID)
	assert.NoError(t, err)
	assert.Len(t, got, 1)

	for _, nb := range got {
		_, ok := exp[nb.NotebookID]
		assert.True(t, ok)
	}

	exp = map[string]*Notebook{
		nbs[1].NotebookID: nbs[1],
	}
	got, err = st.ListNotebooksByState(NotebookStateRequeued)
	assert.NoError(t, err)
	assert.Len(t, got, 1)

	for _, nb := range got {
		_, ok := exp[nb.NotebookID]
		assert.True(t, ok)
	}
}

func TestUpdateNotebook(t *testing.T) {
	const (
		name      = "notebook0"
		projectID = "pid0"
	)
	st, teardown := NewTest(t)
	defer teardown()

	nb := &Notebook{
		Name:      name,
		ProjectID: projectID,
		State:     NotebookStateRunning,
	}
	err := st.CreateNotebook(nb)
	assert.NoError(t, err)

	nb0, err := st.GetActiveNotebookByNameAndProjectID(name, projectID)
	assert.NoError(t, err)
	curVersion := nb0.Version

	nb0.State = NotebookStateQueued
	err = st.UpdateNotebookForRescheduling(nb0)
	assert.NoError(t, err)

	nb1, err := st.GetActiveNotebookByNameAndProjectID(name, projectID)
	assert.NoError(t, err)
	assert.Equal(t, curVersion+1, nb1.Version)
	assert.Equal(t, NotebookStateQueued, nb1.State)
}

func TestCountActiveNotebooksByProjectID(t *testing.T) {
	st, teardown := NewTest(t)
	defer teardown()

	nbs := []*Notebook{
		{
			NotebookID: "nb0",
			ProjectID:  "pid0",
			State:      NotebookStateRunning,
		},
		{
			NotebookID: "nb1",
			ProjectID:  "pid0",
			State:      NotebookStateStopped,
		},
		{
			NotebookID: "nb2",
			ProjectID:  "pid0",
			State:      NotebookStateDeleted,
		},
		{
			NotebookID: "nb3",
			ProjectID:  "pid1",
			State:      NotebookStateRunning,
		},
	}
	for _, nb := range nbs {
		err := st.CreateNotebook(nb)
		assert.NoError(t, err)
	}

	count, err := st.CountActiveNotebooksByProjectID("pid0")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)

	count, err = st.CountActiveNotebooksByProjectID("pid1")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	count, err = st.CountActiveNotebooksByProjectID("pid2")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}
