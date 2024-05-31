package store

import (
	"fmt"

	v1 "github.com/llm-operator/job-manager/api/v1"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// NotebookState is the state of a notebook.
type NotebookState string

const (
	// NotebookStateQueued is the state of a notebook that is waiting to be scheduled.
	NotebookStateQueued NotebookState = "queued"
	// NotebookStateRunning is the state of a notebook that is currently running.
	NotebookStateRunning NotebookState = "running"
	// NotebookStateStopped is the state of a notebook that has been stopped.
	NotebookStateStopped NotebookState = "stopped"
	// NotebookStateFailed is the state of a notebook that has failed.
	NotebookStateFailed NotebookState = "failed"
)

// NotebookQueuedAction is the action of a queued notebook.
type NotebookQueuedAction string

const (
	// NotebookQueuedActionStart is the action to start a notebook.
	NotebookQueuedActionStart NotebookQueuedAction = "queued"
	// NotebookQueuedActionStop is the action to stop a notebook.
	NotebookQueuedActionStop NotebookQueuedAction = "stopping"
	// NotebookQueuedActionDelete is the action to delete a notebook.
	NotebookQueuedActionDelete NotebookQueuedAction = "deleting"
)

// Notebook is a model of notebook.
type Notebook struct {
	gorm.Model

	NotebookID string `gorm:"uniqueIndex"`

	Image string

	// Message is the marshalled JSON of the v1.Notebook.
	Message []byte

	State        NotebookState
	QueuedAction NotebookQueuedAction

	TenantID            string
	OrganizationID      string
	ProjectID           string `gorm:"index"`
	KubernetesNamespace string

	Version int
}

// V1Notebook converts a notebook to a v1.Notebook.
func (n *Notebook) V1Notebook() (*v1.Notebook, error) {
	var nbProto v1.Notebook
	if err := proto.Unmarshal(n.Message, &nbProto); err != nil {
		return nil, err
	}
	if n.State == NotebookStateQueued {
		nbProto.Status = string(n.QueuedAction)
	} else {
		nbProto.Status = string(n.State)
	}
	return &nbProto, nil
}

// MutateMessage mutates the message field of a notebook.
func (n *Notebook) MutateMessage(mutateFn func(nb *v1.Notebook)) error {
	nbProto, err := n.V1Notebook()
	if err != nil {
		return err
	}
	mutateFn(nbProto)
	msg, err := proto.Marshal(nbProto)
	if err != nil {
		return err
	}
	n.Message = msg
	return nil
}

// CreateNotebook creates a new notebook.
func (s *S) CreateNotebook(nb *Notebook) error {
	return s.db.Create(nb).Error
}

// GetNotebookByIDAndProjectID gets a notebook by its notebook ID and tenant ID.
func (s *S) GetNotebookByIDAndProjectID(id, projectID string) (*Notebook, error) {
	var nb Notebook
	if err := s.db.Where("notebook_id = ? AND project_id = ?", id, projectID).Take(&nb).Error; err != nil {
		return nil, err
	}
	return &nb, nil
}

// ListNotebooksByProjectIDWithPagination finds notebooks with pagination. Notebooks are returned with a descending order of ID.
func (s *S) ListNotebooksByProjectIDWithPagination(projectID string, afterID uint, limit int) ([]*Notebook, bool, error) {
	var nbs []*Notebook
	q := s.db.Where("project_id = ?", projectID)
	if afterID > 0 {
		q = q.Where("id < ?", afterID)
	}
	if err := q.Order("id DESC").Limit(limit + 1).Find(&nbs).Error; err != nil {
		return nil, false, err
	}

	var hasMore bool
	if len(nbs) > limit {
		nbs = nbs[:limit]
		hasMore = true
	}
	return nbs, hasMore, nil
}

// ListQueuedNotebooks finds queued notebooks.
func (s *S) ListQueuedNotebooks() ([]*Notebook, error) {
	var nbs []*Notebook
	if err := s.db.Where("state = ?", NotebookStateQueued).Find(&nbs).Error; err != nil {
		return nil, err
	}
	return nbs, nil
}

// SetNotebookQueuedAction sets a notebook queued action.
func (s *S) SetNotebookQueuedAction(id string, currentVersion int, newAction NotebookQueuedAction) error {
	result := s.db.Model(&Notebook{}).
		Where("notebook_id = ?", id).
		Where("version = ?", currentVersion).
		Updates(map[string]interface{}{
			"state":         NotebookStateQueued,
			"queued_action": newAction,
			"version":       currentVersion + 1,
		})
	if err := result.Error; err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("update notebook: %w", ErrConcurrentUpdate)
	}
	return nil
}

// SetNonQueuedStateAndMessage sets a non-queued state and message.
func (s *S) SetNonQueuedStateAndMessage(id string, currentVersion int, newState NotebookState, message []byte) error {
	result := s.db.Model(&Notebook{}).
		Where("notebook_id = ?", id).
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
		return fmt.Errorf("update notebook: %w", ErrConcurrentUpdate)
	}
	return nil
}

// DeleteNotebook deletes a notebook.
func (s *S) DeleteNotebook(id, projectID string) error {
	res := s.db.Unscoped().Where("notebook_id = ? AND project_id = ?", id, projectID).Delete(&Notebook{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
