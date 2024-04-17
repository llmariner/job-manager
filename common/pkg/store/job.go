package store

import (
	"fmt"

	"gorm.io/gorm"
)

// JobState represents the state of a job.
type JobState string

const (
	// JobStateQueued represents the pending state.
	JobStateQueued JobState = "queued"
	// JobStateRunning represents the running state.
	JobStateRunning JobState = "running"
	// JobStatusFailed represents the failed state.
	JobStatusFailed JobState = "failed"
	// JobStateSucceeded represents the succeeded state.
	JobStateSucceeded JobState = "succeeded"
	// JobStateCancelled represents the cancelled state.
	JobStateCancelled JobState = "cancelled"
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

	// OutputModelID is the ID of a generated model.
	OutputModelID string

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

// ListQueuedJobs finds queued jobs.
func (s *S) ListQueuedJobs() ([]*Job, error) {
	var jobs []*Job
	if err := s.db.Where("state = ?", JobStateQueued).Order("job_id").Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// ListQueuedJobsByTenantID finds queued jobs.
func (s *S) ListQueuedJobsByTenantID(tenantID string) ([]*Job, error) {
	var jobs []*Job
	if err := s.db.Where("tenant_id = ? AND state = ?", tenantID, JobStateQueued).Order("job_id").Find(&jobs).Error; err != nil {
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

// UpdateOutputModelID updates the output model ID.
func (s *S) UpdateOutputModelID(jobID string, currentVersion int, outputModelID string) error {
	result := s.db.Model(&Job{}).
		Where("job_id = ?", jobID).
		Where("version = ?", currentVersion).
		Updates(map[string]interface{}{
			"output_model_id": outputModelID,
			"version":         currentVersion + 1,
		})
	if err := result.Error; err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("update job: %w", ErrConcurrentUpdate)
	}
	return nil
}
