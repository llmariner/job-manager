package store

import (
	"fmt"

	"gorm.io/gorm"
)

// JobState represents the state of a job.
type JobState string

const (
	// JobStatePending represents the pending state.
	// TODO(kenji): Consider renaming this to "queued" to be consistent with OpenAI API.
	JobStatePending JobState = "pending"
	// JobStateRunning represents the running state.
	JobStateRunning JobState = "running"
	// JobStateCompleted represents the completed state.
	// TODO(kenji): Consider renaming this to "succeeded" or "failed" to be consistent with OpenAI API.
	JobStateCompleted JobState = "completed"
)

// Job represents a job.
type Job struct {
	gorm.Model

	JobID string `gorm:"uniqueIndex:idx_job_job_id"`

	// Message is the marshaled proto message of v1.Job.
	Message []byte

	// Suffix is a string that will be added to a fine-tuned model name.
	Suffix string

	State    JobState `gorm:"index:idx_job_state_tenant_id"`
	TenantID string   `gorm:"index:idx_job_state_tenant_id"`

	Version int
}

// CreateJob creates a new job.
func (s *S) CreateJob(job *Job) error {
	if err := s.db.Create(job).Error; err != nil {
		return err
	}
	return nil
}

// GetJobByJobID gets a job.
func (s *S) GetJobByJobID(jobID string) (*Job, error) {
	var job Job
	if err := s.db.Where("job_id = ?", jobID).Take(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// ListPendingJobs finds pending jobs.
func (s *S) ListPendingJobs() ([]*Job, error) {
	var jobs []*Job
	if err := s.db.Where("state = ?", JobStatePending).Order("job_id").Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// ListPendingJobsByTenantID finds pending jobs.
func (s *S) ListPendingJobsByTenantID(tenantID string) ([]*Job, error) {
	var jobs []*Job
	if err := s.db.Where("tenant_id = ? AND state = ?", tenantID, JobStatePending).Order("job_id").Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// ListJobsByTenantID finds jobs.
func (s *S) ListJobsByTenantID(tenantID string) ([]*Job, error) {
	var jobs []*Job
	if err := s.db.Where("tenant_id = ?", tenantID).Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// UpdateJobState updates a job.
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
