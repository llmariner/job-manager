package store

import (
	"fmt"

	"gorm.io/gorm"
)

// JobState represents the state of a job.
type JobState string

const (
	// JobStatePending represents the pending state.
	JobStatePending JobState = "pending"
	// JobStateRunning represents the running state.
	JobStateRunning JobState = "running"
	// JobStateCompleted represents the completed state.
	JobStateCompleted JobState = "completed"
)

type Job struct {
	gorm.Model

	JobID string `gorm:"uniqueIndex:idx_job_job_id"`

	// Message is the marshaled proto message of v1.Job.
	Message []byte

	TenantID string   `gorm:"index:idx_job_tenant_id_state"`
	State    JobState `gorm:"index:idx_job_tenant_id_state"`

	Version int
}

// CreateJob creates a new job.
func (s *S) CreateJob(job *Job) error {
	if err := s.db.Create(job).Error; err != nil {
		return err
	}
	return nil
}

// FindPendingJobs finds peending jobs.
func (s *S) FindPendingJobs(tenantID string) ([]*Job, error) {
	var jobs []*Job
	if err := s.db.Where("tenant_id = ? AND state = ?", tenantID, JobStatePending).Order("job_id").Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// FindJobs finds jobs.
func (s *S) FindJobs(tenantID string) ([]*Job, error) {
	var jobs []*Job
	if err := s.db.Where("tenant_id = ?", tenantID).Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// UpdateJob updates a job.
func (s *S) UpdateJobState(jobID string, currentVersion int, newState JobState) error {
	result := s.db.Model(&Job{}).
		Where("job_id = ?", jobID).
		Where("version = ?", currentVersion).
		Updates(map[string]interface{}{
			"state":   newState,
			"version": currentVersion + 1,
		})
	if err := result.Error; err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("update job: %w", ErrConcurrentUpdate)
	}
	return nil
}
