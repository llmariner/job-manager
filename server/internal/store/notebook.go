package store

import (
	"fmt"
	"strings"

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
	// NotebookStateDeleted is the state of a notebook that has been deleted.
	NotebookStateDeleted NotebookState = "deleted"
)

// NotebookQueuedAction is the action of a queued notebook.
type NotebookQueuedAction string

const (
	// NotebookQueuedActionStart is the action to start a notebook.
	NotebookQueuedActionStart NotebookQueuedAction = "starting"
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

	State NotebookState
	// QueuedAction is the action of the queued notebook. This field is only used when
	// the state is NotebookStateQueued, and processed by the dispatcher.
	QueuedAction NotebookQueuedAction

	TenantID            string
	OrganizationID      string
	ProjectID           string `gorm:"index:idx_notebook_project_id_name"`
	KubernetesNamespace string
	// ClusterID is the ID of the cluster where the job runs.
	ClusterID string

	// We do not use a unique index here since the same notebook name can be used if there is only one active noteobook.
	Name string `gorm:"index:idx_notebook_project_id_name"`

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

// V1InternalNotebook converts a notebook to a v1.InternalNotebook.
func (n *Notebook) V1InternalNotebook() (*v1.InternalNotebook, error) {
	nbProto, err := n.V1Notebook()
	if err != nil {
		return nil, err
	}
	state, err := convertToV1NotebookState(n.State)
	if err != nil {
		return nil, err
	}
	action, err := convertToNotebookQueuedAction(n.QueuedAction)
	if err != nil {
		return nil, err
	}
	return &v1.InternalNotebook{
		Notebook:     nbProto,
		State:        state,
		QueuedAction: action,
	}, nil
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

func convertToV1NotebookState(state NotebookState) (v1.NotebookState, error) {
	v, ok := v1.NotebookState_value[strings.ToUpper(string(state))]
	if !ok {
		return 0, fmt.Errorf("unknown notebook state: %s", state)
	}
	return v1.NotebookState(v), nil
}

func convertToNotebookQueuedAction(action NotebookQueuedAction) (v1.NotebookQueuedAction, error) {
	if action == "" {
		// when the status is not queued, the queued action is not set.
		return v1.NotebookQueuedAction_ACTION_UNSPECIFIED, nil
	}
	v, ok := v1.NotebookQueuedAction_value[strings.ToUpper(string(action))]
	if !ok {
		return 0, fmt.Errorf("unknown notebook queued action: %s", action)
	}
	return v1.NotebookQueuedAction(v), nil
}

// CreateNotebook creates a new notebook.
func (s *S) CreateNotebook(nb *Notebook) error {
	return s.db.Create(nb).Error
}

// GetNotebookByID gets a notebook by its notebook ID.
func (s *S) GetNotebookByID(id string) (*Notebook, error) {
	var nb Notebook
	if err := s.db.Where("notebook_id = ?", id).Take(&nb).Error; err != nil {
		return nil, err
	}
	return &nb, nil
}

// GetNotebookByIDAndProjectID gets a notebook by its notebook ID and project ID.
func (s *S) GetNotebookByIDAndProjectID(id, projectID string) (*Notebook, error) {
	var nb Notebook
	if err := s.db.Where("notebook_id = ? AND project_id = ?", id, projectID).Take(&nb).Error; err != nil {
		return nil, err
	}
	return &nb, nil
}

// GetActiveNotebookByNameAndProjectID gets an active notebook by its name and project ID.
func (s *S) GetActiveNotebookByNameAndProjectID(name, projectID string) (*Notebook, error) {
	var nb Notebook
	if err := s.db.Where("name = ? AND project_id = ? AND state != ?", name, projectID, NotebookStateDeleted).Take(&nb).Error; err != nil {
		return nil, err
	}
	return &nb, nil
}

// ListActiveNotebooksByProjectIDWithPagination finds active notebooks with pagination. Notebooks are returned with a descending order of ID.
func (s *S) ListActiveNotebooksByProjectIDWithPagination(projectID string, afterID uint, limit int) ([]*Notebook, bool, error) {
	var nbs []*Notebook
	q := s.db.Where("project_id = ? AND state != ?", projectID, NotebookStateDeleted)
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

// ListQueuedNotebooksByTenantID finds queued notebooks by tenant ID.
func (s *S) ListQueuedNotebooksByTenantID(tenantID string) ([]*Notebook, error) {
	var nbs []*Notebook
	if err := s.db.Where("tenant_id = ? AND state = ?", tenantID, NotebookStateQueued).Find(&nbs).Error; err != nil {
		return nil, err
	}
	return nbs, nil
}

// SetNotebookQueuedAction sets a notebook queued action.
func (s *S) SetNotebookQueuedAction(id string, currentVersion int, newAction NotebookQueuedAction) (*Notebook, error) {
	var nb Notebook
	result := s.db.Model(&nb).
		Where("notebook_id = ?", id).
		Where("version = ?", currentVersion).
		Updates(map[string]interface{}{
			"state":         NotebookStateQueued,
			"queued_action": newAction,
			"version":       currentVersion + 1,
		})
	if err := result.Error; err != nil {
		return nil, err
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("update notebook: %w", ErrConcurrentUpdate)
	}
	return &nb, nil
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
