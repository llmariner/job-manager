package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCountActiveBatchJobsByProjectID(t *testing.T) {
	st, teardown := NewTest(t)
	defer teardown()

	jobs := []*BatchJob{
		{
			JobID:     "job0",
			ProjectID: "pid0",
			State:     BatchJobStateRunning,
		},
		{
			JobID:     "job1",
			ProjectID: "pid0",
			State:     BatchJobStateSucceeded,
		},
		{
			JobID:     "job2",
			ProjectID: "pid0",
			State:     BatchJobStateDeleted,
		},
		{
			JobID:     "job3",
			ProjectID: "pid1",
			State:     BatchJobStateRunning,
		},
	}
	for _, job := range jobs {
		err := st.CreateBatchJob(job)
		assert.NoError(t, err)
	}

	count, err := st.CountActiveBatchJobsByProjectID("pid0")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)

	count, err = st.CountActiveBatchJobsByProjectID("pid1")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	count, err = st.CountActiveBatchJobsByProjectID("pid2")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}
