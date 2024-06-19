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
