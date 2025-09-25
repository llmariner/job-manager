package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/llmariner/common/pkg/aws"
	v1 "github.com/llmariner/job-manager/api/v1"
	rbacv1 "github.com/llmariner/rbac-manager/api/v1"
	"github.com/llmariner/rbac-manager/pkg/auth"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// NotebookState is the state of a notebook.
type NotebookState string

const (
	// NotebookStateQueued is the state of a notebook that is waiting to be scheduled.
	NotebookStateQueued NotebookState = "queued"
	// NotebookStateInitializing is the state of a notebook that is initializing.
	NotebookStateInitializing NotebookState = "initializing"
	// NotebookStateRunning is the state of a notebook that is currently running.
	NotebookStateRunning NotebookState = "running"
	// NotebookStateStopped is the state of a notebook that has been stopped.
	NotebookStateStopped NotebookState = "stopped"
	// NotebookStateFailed is the state of a notebook that has failed.
	NotebookStateFailed NotebookState = "failed"
	// NotebookStateDeleted is the state of a notebook that has been deleted.
	NotebookStateDeleted NotebookState = "deleted"
	// NotebookStateRequeued is the state of a notebook that has been requeued from unavailable clusters.
	NotebookStateRequeued NotebookState = "requeued"
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
	// NotebookQueuedActionRequeue is the action to requeue a notebook.
	NotebookQueuedActionRequeue NotebookQueuedAction = "requeueing"
)

// Notebook is a model of notebook.
type Notebook struct {
	gorm.Model

	// Use unix nano seconds as creating time, so we can truncate the timestamp
	// to any specified duration (e.g., hour, day) easily in the query to summarize
	// notebook job stats.
	Created int64 `gorm:"autoCreateTime:nano"`

	NotebookID string `gorm:"uniqueIndex"`

	// We do not use a unique index here since the same notebook name can be used if there is only one active noteobook.
	Name string `gorm:"index"`

	ProjectID string `gorm:"index:idx_notebook_project_id_state"`

	TenantID  string `gorm:"index:idx_notebook_tenant_id_cluster_id_state"`
	ClusterID string `gorm:"index:idx_notebook_tenant_id_cluster_id_state"`

	// Message is the marshalled JSON of the v1.Notebook.
	Message []byte
	// ProjectMessage is the marshalled JSON of the rbac v1.Project.
	ProjectMessage []byte

	// APIKey and Token are set when kms encryption is disabled.
	APIKey string
	Token  string
	// EncryptedAPIKey and EncryptedToken are encrypted by data key, and set when kms encryption is enabled.
	EncryptedAPIKey []byte
	EncryptedToken  []byte

	State NotebookState `gorm:"index:idx_notebook_project_id_state;index:idx_notebook_tenant_id_cluster_id_state"`
	// QueuedAction is the action of the queued notebook. This field is only used when
	// the state is NotebookStateQueued, and processed by the dispatcher.
	QueuedAction NotebookQueuedAction
	// Reason explains why the notebook is in the current state
	Reason string

	Version int
}

// GetAPIKey returns the API key, decrypting it if necessary.
func (n *Notebook) GetAPIKey(ctx context.Context, dataKey []byte) (string, error) {
	if len(dataKey) > 0 && len(n.EncryptedAPIKey) > 0 {
		// Use the AWS decrypt function with notebook ID as context
		decrypted, err := aws.Decrypt(ctx, n.EncryptedAPIKey, n.NotebookID, dataKey)
		if err != nil {
			return "", fmt.Errorf("decrypt api key: %w", err)
		}
		return decrypted, nil
	}
	return n.APIKey, nil
}

// GetToken returns the token, decrypting it if necessary.
func (n *Notebook) GetToken(ctx context.Context, dataKey []byte) (string, error) {
	if len(dataKey) > 0 && len(n.EncryptedToken) > 0 {
		// Use the AWS decrypt function with notebook ID as context
		decrypted, err := aws.Decrypt(ctx, n.EncryptedToken, n.NotebookID, dataKey)
		if err != nil {
			return "", fmt.Errorf("decrypt token: %w", err)
		}
		return decrypted, nil
	}
	return n.Token, nil
}

// SetAPIKey sets the API key, encrypting it if a data key is provided.
func (n *Notebook) SetAPIKey(ctx context.Context, apiKey string, dataKey []byte) error {
	if len(dataKey) > 0 {
		// Use the AWS encrypt function with notebook ID as context
		encrypted, err := aws.Encrypt(ctx, apiKey, n.NotebookID, dataKey)
		if err != nil {
			return fmt.Errorf("encrypt api key: %w", err)
		}
		n.EncryptedAPIKey = encrypted
		n.APIKey = ""
	} else {
		n.APIKey = apiKey
	}
	return nil
}

// SetToken sets the token, encrypting it if a data key is provided.
func (n *Notebook) SetToken(ctx context.Context, token string, dataKey []byte) error {
	if len(dataKey) > 0 {
		// Use the AWS encrypt function with notebook ID as context
		encrypted, err := aws.Encrypt(ctx, token, n.NotebookID, dataKey)
		if err != nil {
			return fmt.Errorf("encrypt token: %w", err)
		}
		n.EncryptedToken = encrypted
		n.Token = ""
	} else {
		n.Token = token
	}
	return nil
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

// RebuildUserInfo rebuilds the user info from the project message.
func (n *Notebook) RebuildUserInfo() (*auth.UserInfo, error) {
	var pProto rbacv1.Project
	err := proto.Unmarshal(n.ProjectMessage, &pProto)
	if err != nil {
		return nil, err
	}
	var akes []auth.AssignedKubernetesEnv
	for _, a := range pProto.AssignedKubernetesEnvs {
		akes = append(akes, auth.AssignedKubernetesEnv{
			ClusterID:   a.ClusterId,
			ClusterName: a.ClusterName,
			Namespace:   a.Namespace,
		})
	}
	return &auth.UserInfo{
		AssignedKubernetesEnvs: akes,
		TenantID:               n.TenantID,
		ProjectID:              n.ProjectID,
	}, nil
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

// ListQueuedNotebooksByTenantIDAndClusterID finds queued notebooks by tenant ID and cluster ID.
func (s *S) ListQueuedNotebooksByTenantIDAndClusterID(tenantID, clusterID string) ([]*Notebook, error) {
	var nbs []*Notebook
	if err := s.db.Where("tenant_id = ? AND cluster_id = ? AND state = ?", tenantID, clusterID, NotebookStateQueued).
		Find(&nbs).Error; err != nil {
		return nil, err
	}
	return nbs, nil
}

// ListNotebooksByState finds all notebooks with the specified state.
func (s *S) ListNotebooksByState(state NotebookState) ([]*Notebook, error) {
	var nbs []*Notebook
	if err := s.db.Where("state = ?", state).Find(&nbs).Error; err != nil {
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
			"reason":        "",
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

// SetState sets a state.
func (s *S) SetState(id string, currentVersion int, newState NotebookState) error {
	result := s.db.Model(&Notebook{}).
		Where("notebook_id = ?", id).
		Where("version = ?", currentVersion).
		Updates(map[string]interface{}{
			"state":   newState,
			"reason":  "",
			"version": currentVersion + 1,
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
func (s *S) SetNonQueuedStateAndMessage(id string, currentVersion int, newState NotebookState, message []byte, reason string) error {
	result := s.db.Model(&Notebook{}).
		Where("notebook_id = ?", id).
		Where("version = ?", currentVersion).
		Updates(map[string]interface{}{
			"state":         newState,
			"queued_action": "",
			"message":       message,
			"reason":        reason,
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

// UpdateNotebookForRescheduling updates the notebook.
func (s *S) UpdateNotebookForRescheduling(nb *Notebook) error {
	result := s.db.Model(&Notebook{}).
		Where("notebook_id = ?", nb.NotebookID).
		Where("version = ?", nb.Version).
		Updates(map[string]interface{}{
			"cluster_id":    nb.ClusterID,
			"state":         nb.State,
			"queued_action": nb.QueuedAction,
			"message":       nb.Message,
			"reason":        nb.Reason,
			"version":       nb.Version + 1,
		})
	if err := result.Error; err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("update notebook: %w", ErrConcurrentUpdate)
	}
	return nil
}

// CountActiveNotebooksByProjectID counts the total number of active notebooks by project ID.
func (s *S) CountActiveNotebooksByProjectID(projectID string) (int64, error) {
	var count int64
	if err := s.db.Model(&Notebook{}).Where("project_id = ? AND state != ?", projectID, NotebookStateDeleted).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
