package store

import (
	"fmt"

	v1 "github.com/llm-operator/job-manager/api/v1"
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
)

// BatchJobQueuedAction is the action of a queue batch job.
type BatchJobQueuedAction string

const (
	// BatchJobQueuedActionCreate is the action to create a batch job.
	BatchJobQueuedActionCreate BatchJobQueuedAction = "creating"
	// BatchJobQueuedActionCancel is the action to cancel a batch job.
	BatchJobQueuedActionCancel BatchJobQueuedAction = "canceling"
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

	TenantID            string
	OrganizationID      string
	ProjectID           string
	KubernetesNamespace string

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

// CreateBatchJob creates a batch job.
func (s *S) CreateBatchJob(job *BatchJob) error {
	return s.db.Create(job).Error
}

// GetBatchJobByIDAndProjectID gets a batch job by its job ID and project ID.
func (s *S) GetBatchJobByIDAndProjectID(id, projectID string) (*BatchJob, error) {
	var job BatchJob
	if err := s.db.Where("job_id = ? AND project_id = ?", id, projectID).Take(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// ListBatchJobsByProjectIDWithPagination lists batch jobs by project ID with pagination.
func (s *S) ListBatchJobsByProjectIDWithPagination(projectID string, afterID uint, limit int) ([]BatchJob, bool, error) {
	var jobs []BatchJob
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
