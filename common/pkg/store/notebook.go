package store

import (
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
	ProjectID           string
	KubernetesNamespace string

	Version int
}

// CreateNotebook creates a new notebook.
func (s *S) CreateNotebook(nb *Notebook) error {
	return s.db.Create(nb).Error
}
