package store

import (
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

// Notebook is a model of notebook.
type Notebook struct {
	gorm.Model

	NotebookID string `gorm:"uniqueIndex"`

	Image string

	// Message is the marshalled JSON of the v1.Notebook.
	Message []byte

	State NotebookState

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
	nbProto.Status = string(n.State)
	return &nbProto, nil
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
