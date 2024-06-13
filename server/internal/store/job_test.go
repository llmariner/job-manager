package store

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestCreateAndGetJob(t *testing.T) {
	st, teardown := NewTest(t)
	defer teardown()

	_, err := st.GetJobByJobID("job0")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))

	job := &Job{
		JobID:     "job0",
		State:     JobStateQueued,
		ProjectID: "pid0",
	}
	err = st.CreateJob(job)
	assert.NoError(t, err)

	got, err := st.GetJobByJobID("job0")
	assert.NoError(t, err)
	assert.Equal(t, job.JobID, got.JobID)

	got, err = st.GetJobByJobIDAndProjectID("job0", "pid0")
	assert.NoError(t, err)
	assert.Equal(t, job.JobID, got.JobID)

	// Different tenant.
	_, err = st.GetJobByJobIDAndProjectID("job0", "pid1")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestCreateAndListJobs(t *testing.T) {
	st, teardown := NewTest(t)
	defer teardown()

	jobs := []*Job{
		{
			JobID:    "job0",
			State:    JobStateQueued,
			TenantID: "tid0",
		},
		{
			JobID:    "job1",
			State:    JobStateRunning,
			TenantID: "tid0",
		},
		{
			JobID:    "job2",
			State:    JobStateQueued,
			TenantID: "tid1",
		},
	}
	for _, job := range jobs {
		err := st.CreateJob(job)
		assert.NoError(t, err)
	}

	got, err := st.ListQueuedJobs()
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, jobs[0].JobID, got[0].JobID)
	assert.Equal(t, jobs[2].JobID, got[1].JobID)

	got, err = st.ListQueuedJobsByTenantID("tid0")
	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, jobs[0].JobID, got[0].JobID)

	got, err = st.ListQueuedJobsByTenantID("tid1")
	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, jobs[2].JobID, got[0].JobID)

	got, err = st.ListJobsByTenantID("tid0")
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, jobs[0].JobID, got[0].JobID)
	assert.Equal(t, jobs[1].JobID, got[1].JobID)
}

func TestListJobsByProjectIDWithPagination(t *testing.T) {
	st, teardown := NewTest(t)
	defer teardown()

	for i := 0; i < 10; i++ {
		job := &Job{
			JobID:     fmt.Sprintf("job%d", i),
			State:     JobStateQueued,
			ProjectID: "pid0",
		}
		err := st.CreateJob(job)
		assert.NoError(t, err)
	}

	got, hasMore, err := st.ListJobsByProjectIDWithPagination("pid0", 0, 5)
	assert.NoError(t, err)
	assert.True(t, hasMore)
	assert.Len(t, got, 5)
	want := []string{"job9", "job8", "job7", "job6", "job5"}
	for i, job := range got {
		assert.Equal(t, want[i], job.JobID)
	}

	got, hasMore, err = st.ListJobsByProjectIDWithPagination("pid0", got[4].ID, 2)
	assert.NoError(t, err)
	assert.True(t, hasMore)
	assert.Len(t, got, 2)
	want = []string{"job4", "job3"}
	for i, job := range got {
		assert.Equal(t, want[i], job.JobID)
	}

	got, hasMore, err = st.ListJobsByProjectIDWithPagination("pid0", got[1].ID, 3)
	assert.NoError(t, err)
	assert.False(t, hasMore)
	assert.Len(t, got, 3)
	want = []string{"job2", "job1", "job0"}
	for i, job := range got {
		assert.Equal(t, want[i], job.JobID)
	}
}

func TestUpdateJobState(t *testing.T) {
	st, teardown := NewTest(t)
	defer teardown()

	job := &Job{
		JobID:   "job0",
		State:   JobStateQueued,
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

func TestUpdateOutputModelID(t *testing.T) {
	st, teardown := NewTest(t)
	defer teardown()

	job := &Job{
		JobID:   "job0",
		State:   JobStateQueued,
		Version: 1,
	}
	err := st.CreateJob(job)
	assert.NoError(t, err)

	err = st.UpdateOutputModelID(job.JobID, job.Version, "output-model-id")
	assert.NoError(t, err)

	got, err := st.GetJobByJobID("job0")
	assert.NoError(t, err)
	assert.Equal(t, "output-model-id", got.OutputModelID)
}
