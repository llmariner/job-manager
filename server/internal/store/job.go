package store

import (
	"fmt"
	"strings"

	v1 "github.com/llmariner/job-manager/api/v1"
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
	// JobStateFailed represents the failed state.
	JobStateFailed JobState = "failed"
	// JobStateSucceeded represents the succeeded state.
	JobStateSucceeded JobState = "succeeded"
	// JobStateCanceled represents the canceled state.
	JobStateCanceled JobState = "canceled"
)

// JobQueuedAction is the action of a queue job.
type JobQueuedAction string

const (
	// JobQueuedActionCreate represents the creating action.
	JobQueuedActionCreate JobQueuedAction = "creating"
	// JobQueuedActionCancel represents the canceling action.
	JobQueuedActionCancel JobQueuedAction = "canceling"
)

// Job represents a job.
type Job struct {
	gorm.Model

	JobID string `gorm:"uniqueIndex:idx_job_job_id"`

	// Message is the marshaled proto message of v1.Job.
	Message []byte

	// Suffix is a string that will be added to a fine-tuned model name.
	Suffix string

	// QueuedAction is the action of a queue job.
	// This field is only used when the state is JobStateQueued.
	QueuedAction JobQueuedAction

	State    JobState `gorm:"index:idx_job_state_tenant_id,idx_job_tenant_id_cluster_id"`
	TenantID string   `gorm:"index:idx_job_state_tenant_id"`

	OrganizationID string
	ProjectID      string `gorm:"index"`
	ClusterID      string `gorm:"index:idx_job_tenant_id_cluster_id"`

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
	if j.State == JobStateQueued {
		jobProto.Status = string(j.QueuedAction)
	} else {
		jobProto.Status = string(j.State)
	}
	return &jobProto, nil
}

// V1InternalJob converts a job to v1.InternalJob.
func (j *Job) V1InternalJob() (*v1.InternalJob, error) {
	job, err := j.V1Job()
	if err != nil {
		return nil, err
	}
	state, err := convertToV1JobState(j.State)
	if err != nil {
		return nil, err
	}
	action, err := convertToV1JobQueuedAction(j.QueuedAction)
	if err != nil {
		return nil, err
	}
	return &v1.InternalJob{
		Job:           job,
		OutputModelId: j.OutputModelID,
		Suffix:        j.Suffix,
		State:         state,
		QueuedAction:  action,
	}, nil
}

func convertToV1JobState(state JobState) (v1.InternalJob_State, error) {
	v, ok := v1.InternalJob_State_value[strings.ToUpper(string(state))]
	if !ok {
		return v1.InternalJob_STATE_UNSPECIFIED, fmt.Errorf("unknown job state: %s", state)
	}
	return v1.InternalJob_State(v), nil
}

func convertToV1JobQueuedAction(action JobQueuedAction) (v1.InternalJob_Action, error) {
	if action == "" {
		// when the action is not specified, it is considered as unspecified.
		return v1.InternalJob_ACTION_UNSPECIFIED, nil
	}
	v, ok := v1.InternalJob_Action_value[strings.ToUpper(string(action))]
	if !ok {
		return 0, fmt.Errorf("unknown job queued action: %s", action)
	}
	return v1.InternalJob_Action(v), nil
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

// GetJobByJobIDAndProjectID gets a job by its job ID and project ID.
func (s *S) GetJobByJobIDAndProjectID(jobID, projectID string) (*Job, error) {
	var job Job
	if err := s.db.Where("job_id = ? AND project_id = ?", jobID, projectID).Take(&job).Error; err != nil {
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

// ListQueuedJobsByTenantIDAndClusterID finds queued jobs.
func (s *S) ListQueuedJobsByTenantIDAndClusterID(tenantID, clusterID string) ([]*Job, error) {
	var jobs []*Job
	if err := s.db.Where("tenant_id = ? AND cluster_id = ? AND state = ?", tenantID, clusterID, JobStateQueued).Order("job_id").Find(&jobs).Error; err != nil {
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

// ListJobsByProjectIDWithPagination finds jobs with pagination. Jobs are returned with a descending order of ID.
func (s *S) ListJobsByProjectIDWithPagination(projectID string, afterID uint, limit int) ([]*Job, bool, error) {
	var jobs []*Job
	q := s.db.Where("project_id = ?", projectID)
	if afterID > 0 {
		q = q.Where("id < ?", afterID)
	}
	if err := q.Order("id DESC").Limit(limit + 1).Find(&jobs).Error; err != nil {
		return nil, false, err
	}

	var hasMore bool
	if len(jobs) > limit {
		jobs = jobs[:limit]
		hasMore = true
	}
	return jobs, hasMore, nil
}

// UpdateJobState updates a job state and queued action.
func (s *S) UpdateJobState(jobID string, currentVersion int, newState JobState, newAction JobQueuedAction) (*Job, error) {
	var job Job
	result := s.db.Model(&job).
		Where("job_id = ?", jobID).
		Where("version = ?", currentVersion).
		Updates(map[string]interface{}{
			"state":         newState,
			"queued_action": newAction,
			"version":       currentVersion + 1,
		})
	if err := result.Error; err != nil {
		return nil, err
	}

	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("update job: %w", ErrConcurrentUpdate)
	}
	return &job, nil
}

// UpdateJobStateAndMessage updates a job state and message.
func (s *S) UpdateJobStateAndMessage(jobID string, currentVersion int, newState JobState, message []byte) error {
	result := s.db.Model(&Job{}).
		Where("job_id = ?", jobID).
		Where("version = ?", currentVersion).
		Updates(map[string]interface{}{
			"state":         newState,
			"queued_action": "",
			"message":       message,
			"version":       currentVersion + 1,
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
