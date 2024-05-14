package store

import (
	"fmt"

	v1 "github.com/llm-operator/job-manager/api/v1"
	"google.golang.org/protobuf/proto"
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

// V1Job converts a job to v1.Job.
func (j *Job) V1Job() (*v1.Job, error) {
	var jobProto v1.Job
	err := proto.Unmarshal(j.Message, &jobProto)
	if err != nil {
		return nil, err
	}
	jobProto.Status = string(j.State)
	return &jobProto, nil
}

// MutateMessage mutates the message field of a job.
func (j *Job) MutateMessage(mutateFn func(j *v1.Job)) error {
	jobProto, err := j.V1Job()
	if err != nil {
		return err
	}
	mutateFn(jobProto)
	msg, err := proto.Marshal(jobProto)
	if err != nil {
		return err
	}
	j.Message = msg
	return nil
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

// GetJobByJobIDAndTenantID gets a job by its job ID and tenant ID.
func (s *S) GetJobByJobIDAndTenantID(jobID, tenantID string) (*Job, error) {
	var job Job
	if err := s.db.Where("job_id = ? AND tenant_id = ?", jobID, tenantID).Take(&job).Error; err != nil {
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

// UpdateJobStateAndMessage updates a job state and message.
func (s *S) UpdateJobStateAndMessage(jobID string, currentVersion int, newState JobState, message []byte) error {
	result := s.db.Model(&Job{}).
		Where("job_id = ?", jobID).
		Where("version = ?", currentVersion).
		Updates(map[string]interface{}{
			"state":   newState,
			"message": message,
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
