package store

import (
	"fmt"
	"strings"

	v1 "github.com/llmariner/job-manager/api/v1"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// BatchJobState is the state of a batch job.
type BatchJobState string

const (
	// BatchJobStateQueued is the state of a notebook that is waiting to be processed.
	BatchJobStateQueued BatchJobState = "queued"
	// BatchJobStateRunning is the state of a batch job that is currently running.
	BatchJobStateRunning BatchJobState = "running"
	// BatchJobStateSucceeded is the state of a batch job that has completed successfully.
	BatchJobStateSucceeded BatchJobState = "succeeded"
	// BatchJobStateFailed is the state of a batch job that has completed with an error.
	BatchJobStateFailed BatchJobState = "failed"
	// BatchJobStateCanceled is the state of a batch job that has been canceled.
	BatchJobStateCanceled BatchJobState = "canceled"
	// BatchJobStateDeleted is the state of a batch job that has been deleted.
	BatchJobStateDeleted BatchJobState = "deleted"
)

// BatchJobQueuedAction is the action of a queue batch job.
type BatchJobQueuedAction string

const (
	// BatchJobQueuedActionCreate is the action to create a batch job.
	BatchJobQueuedActionCreate BatchJobQueuedAction = "creating"
	// BatchJobQueuedActionCancel is the action to cancel a batch job.
	BatchJobQueuedActionCancel BatchJobQueuedAction = "canceling"
	// BatchJobQueuedActionDelete is the action to delete a batch job.
	BatchJobQueuedActionDelete BatchJobQueuedAction = "deleting"
)

// BatchJob is a model of a batch job.
type BatchJob struct {
	gorm.Model

	JobID string `gorm:"uniqueIndex"`

	Image string
	// Message is the marshaled message of the v1.BatchJob.
	Message []byte

	State BatchJobState
	// QueuedAction is the action of the batch job. This field is only used when
	// the state is BatchJobStateQueued, and processed by the dispatcher.
	QueuedAction BatchJobQueuedAction

	TenantID       string
	OrganizationID string
	ProjectID      string

	Version int
}

// V1BatchJob returns the v1.BatchJob of the batch job.
func (j *BatchJob) V1BatchJob() (*v1.BatchJob, error) {
	var jobProto v1.BatchJob
	if err := proto.Unmarshal(j.Message, &jobProto); err != nil {
		return nil, err
	}
	if j.State == BatchJobStateQueued {
		jobProto.Status = string(j.QueuedAction)
	} else {
		jobProto.Status = string(j.State)
	}
	return &jobProto, nil
}

// V1InternalBatchJob converts a notebook to a v1.InternalBatchJob.
func (j *BatchJob) V1InternalBatchJob() (*v1.InternalBatchJob, error) {
	jobProto, err := j.V1BatchJob()
	if err != nil {
		return nil, err
	}
	state, err := convertToV1BatchJobState(j.State)
	if err != nil {
		return nil, err
	}
	action, err := convertToBatchJobQueuedAction(j.QueuedAction)
	if err != nil {
		return nil, err
	}
	return &v1.InternalBatchJob{
		Job:          jobProto,
		State:        state,
		QueuedAction: action,
	}, nil
}

// MutateMessage mutates the message of the batch job.
func (j *BatchJob) MutateMessage(mutateFn func(*v1.BatchJob)) error {
	jobProto, err := j.V1BatchJob()
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

func convertToV1BatchJobState(state BatchJobState) (v1.InternalBatchJob_State, error) {
	v, ok := v1.InternalBatchJob_State_value[strings.ToUpper(string(state))]
	if !ok {
		return 0, fmt.Errorf("unknown notebook state: %s", state)
	}
	return v1.InternalBatchJob_State(v), nil
}

func convertToBatchJobQueuedAction(action BatchJobQueuedAction) (v1.InternalBatchJob_Action, error) {
	if action == "" {
		// when the status is not queued, the queued action is not set.
		return v1.InternalBatchJob_ACTION_UNSPECIFIED, nil
	}
	v, ok := v1.InternalBatchJob_Action_value[strings.ToUpper(string(action))]
	if !ok {
		return 0, fmt.Errorf("unknown notebook queued action: %s", action)
	}
	return v1.InternalBatchJob_Action(v), nil
}

// CreateBatchJob creates a batch job.
func (s *S) CreateBatchJob(job *BatchJob) error {
	return s.db.Create(job).Error
}

// GetBatchJobByID gets a batch job by its job ID.
func (s *S) GetBatchJobByID(id string) (*BatchJob, error) {
	var job BatchJob
	if err := s.db.Where("job_id = ?", id).Take(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// GetActiveBatchJobByIDAndProjectID gets a batch job by its job ID and project ID.
func (s *S) GetActiveBatchJobByIDAndProjectID(id, projectID string) (*BatchJob, error) {
	var job BatchJob
	if err := s.db.Where("job_id = ? AND project_id = ? AND state != ?", id, projectID, BatchJobStateDeleted).Take(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// ListActiveBatchJobsByProjectIDWithPagination lists batch jobs by project ID with pagination.
func (s *S) ListActiveBatchJobsByProjectIDWithPagination(projectID string, afterID uint, limit int) ([]BatchJob, bool, error) {
	var jobs []BatchJob
	q := s.db.Where("project_id = ? AND state != ?", projectID, BatchJobStateDeleted)
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

// ListQueuedBatchJobsByTenantID finds queued batch jobs by tenant ID.
func (s *S) ListQueuedBatchJobsByTenantID(tenantID string) ([]BatchJob, error) {
	var jobs []BatchJob
	if err := s.db.Where("tenant_id = ? AND state = ?", tenantID, BatchJobStateQueued).Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// SetBatchJobQueuedAction sets the queued action of a batch job.
func (s *S) SetBatchJobQueuedAction(id string, currentVersion int, newActionn BatchJobQueuedAction) (*BatchJob, error) {
	var job BatchJob
	result := s.db.Model(&job).
		Where("job_id = ?", id).
		Where("version = ?", currentVersion).
		Updates(map[string]interface{}{
			"state":         BatchJobStateQueued,
			"queued_action": newActionn,
			"version":       currentVersion + 1,
		})
	if err := result.Error; err != nil {
		return nil, err
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("update batch job: %w", ErrConcurrentUpdate)
	}
	return &job, nil
}

// SetBatchJobState sets the state of a batch job.
func (s *S) SetBatchJobState(id string, currentVersion int, newState BatchJobState) error {
	result := s.db.Model(&BatchJob{}).
		Where("job_id = ?", id).
		Where("version = ?", currentVersion).
		Updates(map[string]interface{}{
			"state":         newState,
			"queued_action": "",
			"version":       currentVersion + 1,
		})
	if err := result.Error; err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("update batch job: %w", ErrConcurrentUpdate)
	}
	return nil
}

// SetNonQueuedBatchJobStateAndMessage sets the state and message of a batch job.
func (s *S) SetNonQueuedBatchJobStateAndMessage(id string, currentVersion int, newState BatchJobState, message []byte) error {
	result := s.db.Model(&BatchJob{}).
		Where("job_id = ?", id).
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
		return fmt.Errorf("update batch job: %w", ErrConcurrentUpdate)
	}
	return nil
}
