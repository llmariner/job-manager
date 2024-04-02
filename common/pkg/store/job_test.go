package store

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateAndFindJob(t *testing.T) {
	st, teardown := NewTest(t)
	defer teardown()

	jobs := []*Job{
		{
			JobID:    "job0",
			State:    JobStatePending,
			TenantID: "tid0",
		},
		{
			JobID:    "job1",
			State:    JobStateRunning,
			TenantID: "tid0",
		},
		{
			JobID:    "job2",
			State:    JobStatePending,
			TenantID: "tid1",
		},
	}
	for _, job := range jobs {
		err := st.CreateJob(job)
		assert.NoError(t, err)
	}

	got, err := st.FindPendingJobs("tid0")
	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, jobs[0].JobID, got[0].JobID)

	got, err = st.FindPendingJobs("tid1")
	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, jobs[2].JobID, got[0].JobID)

	got, err = st.FindJobs("tid0")
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, jobs[0].JobID, got[0].JobID)
	assert.Equal(t, jobs[1].JobID, got[1].JobID)
}

func TestUpdateJobState(t *testing.T) {
	st, teardown := NewTest(t)
	defer teardown()

	job := &Job{
		JobID:   "job0",
		State:   JobStatePending,
		Version: 1,
	}
	err := st.CreateJob(job)
	assert.NoError(t, err)

	err = st.UpdateJobState(job.JobID, job.Version, JobStateRunning)
	assert.NoError(t, err)

	err = st.UpdateJobState(job.JobID, 12345, JobStateRunning)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrConcurrentUpdate))
}
