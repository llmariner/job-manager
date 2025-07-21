package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/llmariner/common/pkg/id"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/server/internal/scheduler"
	"github.com/llmariner/job-manager/server/internal/store"
	rbacv1 "github.com/llmariner/rbac-manager/api/v1"
	"github.com/llmariner/rbac-manager/pkg/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// CreateNotebook creates a notebook.
func (s *S) CreateNotebook(ctx context.Context, req *v1.CreateNotebookRequest) (*v1.Notebook, error) {
	s.logger.Info("Receive CreateNotebook request", "req", req)
	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "failed to extract user info from context")
	}

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	for i, p := range req.AdditionalExposedPorts {
		if p <= 0 {
			return nil, status.Errorf(codes.InvalidArgument, "exposed port[%d] must be greater than 0, but got: %d", i, p)
		}
	}

	if req.Image == nil {
		return nil, status.Error(codes.InvalidArgument, "image is required")
	}
	var image string
	if uri := req.Image.GetUri(); uri != "" {
		image = uri
	} else if t := req.Image.GetType(); t != "" {
		uri, ok := s.nbImageTypes[t]
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "invalid image type: %s (available types: %s)", t, s.nbImageTypeStr)
		}
		image = uri
	} else {
		return nil, status.Error(codes.InvalidArgument, "image uri or type is required")
	}

	// Check if there is any active notebook of the same name.
	// TODO(kenji): Prevent a case where notebooks of the same name are concurrenlty created.
	if _, err := s.store.GetActiveNotebookByNameAndProjectID(req.Name, userInfo.ProjectID); err == nil {
		return nil, status.Errorf(codes.AlreadyExists, "notebook %q already exists", req.Name)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Errorf(codes.Internal, "get notebook: %s", err)
	}

	// TODO(aya): validate resources

	nbID, err := id.GenerateIDForK8SResource("nb-")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate notebook id: %s", err)
	}

	var gpuCount int
	if r := req.Resources; r != nil {
		gpuCount = int(r.GpuCount)
	}

	proj, err := toProjectMessage(userInfo)
	if err != nil {
		return nil, err
	}

	apikey, err := auth.ExtractTokenFromContext(ctx)
	if err != nil {
		return nil, err
	}
	nbToken, err := id.GenerateID("", 48)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate notebook token: %s", err)
	}

	nb := &store.Notebook{
		NotebookID:     nbID,
		ProjectMessage: proj,
		State:          store.NotebookStateQueued,
		QueuedAction:   store.NotebookQueuedActionStart,
		TenantID:       userInfo.TenantID,
		ProjectID:      userInfo.ProjectID,
		Name:           req.Name,
	}

	// Set API key and token using the helper methods
	if err := nb.SetAPIKey(ctx, apikey, s.dataKey); err != nil {
		return nil, status.Errorf(codes.Internal, "set api key: %s", err)
	}

	if err := nb.SetToken(ctx, nbToken, s.dataKey); err != nil {
		return nil, status.Errorf(codes.Internal, "set token: %s", err)
	}

	sresult, err := s.scheduleNotebook(ctx, nb, gpuCount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "schedule: %s", err)
	}
	nb.ClusterID = sresult.ClusterID

	nbProto := &v1.Notebook{
		Id:                  nbID,
		Name:                req.Name,
		CreatedAt:           time.Now().UTC().Unix(),
		Image:               image,
		Resources:           req.Resources,
		Envs:                req.Envs,
		Status:              string(store.NotebookStateQueued),
		ProjectId:           userInfo.ProjectID,
		OrganizationId:      userInfo.OrganizationID,
		KubernetesNamespace: sresult.Namespace,
		ClusterId:           sresult.ClusterID,

		// TODO(kenji): Revisit. We decided to store token here (and in the DB column) for two purposes
		// (support rescheduling & expose the token to the frontend), but it's not clear if it's the best way.
		Token: nbToken,

		OrganizationTitle: userInfo.OrganizationTitle,
		ProjectTitle:      userInfo.ProjectTitle,
		ClusterName:       sresult.ClusterName,

		AdditionalExposedPorts: req.AdditionalExposedPorts,
	}
	msg, err := proto.Marshal(nbProto)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal notebook: %s", err)
	}
	nb.Message = msg

	if err := s.store.CreateNotebook(nb); err != nil {
		return nil, status.Errorf(codes.Internal, "create notebook: %s", err)
	}

	return nbProto, nil
}

func toProjectMessage(userInfo *auth.UserInfo) ([]byte, error) {
	var akesProto []*rbacv1.Project_AssignedKubernetesEnv
	for _, a := range userInfo.AssignedKubernetesEnvs {
		akesProto = append(akesProto, &rbacv1.Project_AssignedKubernetesEnv{
			ClusterId: a.ClusterID,
			Namespace: a.Namespace,
		})
	}
	pProto := &rbacv1.Project{
		Id:                     userInfo.ProjectID,
		AssignedKubernetesEnvs: akesProto,
	}
	proj, err := proto.Marshal(pProto)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal assigned kubernetes env: %s", err)
	}
	return proj, nil
}

func (s *S) scheduleNotebook(ctx context.Context, nb *store.Notebook, gpuCount int) (scheduler.SchedulingResult, error) {
	userInfo, err := nb.RebuildUserInfo()
	if err != nil {
		return scheduler.SchedulingResult{}, status.Errorf(codes.Internal, "rebuild user info: %s", err)
	}

	sresult, err := s.scheduler.Schedule(userInfo, nb.ClusterID, gpuCount)
	if err != nil {
		return sresult, status.Errorf(codes.Internal, "schedule: %s", err)
	}
	if err := s.cache.AddAssumedPod(userInfo.TenantID, sresult.ClusterID,
		fmt.Sprintf("%s/%s", sresult.Namespace, nb.NotebookID), gpuCount); err != nil {
		return sresult, status.Errorf(codes.Internal, "add assumed pod: %s", err)
	}

	// Get the API key and token using the helper methods
	apiKey, err := nb.GetAPIKey(ctx, s.dataKey)
	if err != nil {
		return sresult, status.Errorf(codes.Internal, "get api key: %s", err)
	}

	token, err := nb.GetToken(ctx, s.dataKey)
	if err != nil {
		return sresult, status.Errorf(codes.Internal, "get token: %s", err)
	}

	kclient, err := s.k8sClientFactory.NewClient(sresult.ClusterID, apiKey)
	if err != nil {
		return sresult, status.Errorf(codes.Internal, "create k8s client: %s", err)
	}
	if err := kclient.CreateSecret(ctx, nb.NotebookID, sresult.Namespace, map[string][]byte{
		"OPENAI_API_KEY":    []byte(apiKey),
		"NOTEBOOK_TOKEN":    []byte(token),
		"LLMARINER_API_KEY": []byte(apiKey),
	}); err != nil {
		return sresult, status.Errorf(codes.Internal, "create secret: %s", err)
	}
	return sresult, nil
}

// ListNotebooks lists notebooks.
func (s *S) ListNotebooks(ctx context.Context, req *v1.ListNotebooksRequest) (*v1.ListNotebooksResponse, error) {
	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to extract user info from context")
	}

	if req.Limit < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "limit must be non-negative")
	}
	limit := req.Limit
	if limit == 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}

	var afterID uint
	if req.After != "" {
		nb, err := s.store.GetNotebookByIDAndProjectID(req.After, userInfo.ProjectID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, status.Errorf(codes.InvalidArgument, "invalid after: %s", err)
			}
			return nil, status.Errorf(codes.Internal, "get notebook: %s", err)
		}
		afterID = nb.ID
	}

	nbs, hasMore, err := s.store.ListActiveNotebooksByProjectIDWithPagination(userInfo.ProjectID, afterID, int(limit))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "find notebooks: %s", err)
	}

	var nbProtos []*v1.Notebook
	for _, notebook := range nbs {
		notebookProto, err := notebook.V1Notebook()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "convert notebook to proto: %s", err)
		}
		nbProtos = append(nbProtos, notebookProto)
	}

	totalItems, err := s.store.CountActiveNotebooksByProjectID(userInfo.ProjectID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "count notebooks: %s", err)
	}

	return &v1.ListNotebooksResponse{
		Notebooks:  nbProtos,
		HasMore:    hasMore,
		TotalItems: int32(totalItems),
	}, nil
}

// GetNotebook gets a notebook.
func (s *S) GetNotebook(ctx context.Context, req *v1.GetNotebookRequest) (*v1.Notebook, error) {
	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to extract user info from context")
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	nb, err := s.store.GetNotebookByIDAndProjectID(req.Id, userInfo.ProjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "get notebook: %s", err)
		}
		return nil, status.Errorf(codes.Internal, "get notebook: %s", err)
	}

	nbProto, err := nb.V1Notebook()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "convert notebook to proto: %s", err)
	}
	return nbProto, nil
}

// StopNotebook stops a notebook.
func (s *S) StopNotebook(ctx context.Context, req *v1.StopNotebookRequest) (*v1.Notebook, error) {
	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "failed to extract user info from context")
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	nb, err := s.store.GetNotebookByIDAndProjectID(req.Id, userInfo.ProjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "get notebook: %s", err)
		}
		return nil, status.Errorf(codes.Internal, "get notebook: %s", err)
	}

	nbProto, err := nb.V1Notebook()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "convert notebook to proto: %s", err)
	}

	switch nb.State {
	case store.NotebookStateFailed,
		store.NotebookStateStopped:
		return nbProto, nil
	case store.NotebookStateInitializing,
		store.NotebookStateRunning,
		store.NotebookStateRequeued:
	case store.NotebookStateQueued:
		if nb.QueuedAction == store.NotebookQueuedActionStop ||
			nb.QueuedAction == store.NotebookQueuedActionDelete {
			return nbProto, nil
		}
	default:
		return nil, status.Errorf(codes.Internal, "unknown notebook state: %s", nb.State)
	}

	nb, err = s.store.SetNotebookQueuedAction(nb.NotebookID, nb.Version, store.NotebookQueuedActionStop)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update notebook state: %s", err)
	}
	nbProto, err = nb.V1Notebook()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "convert notebook to proto: %s", err)
	}
	return nbProto, nil
}

// StartNotebook starts a notebook.
func (s *S) StartNotebook(ctx context.Context, req *v1.StartNotebookRequest) (*v1.Notebook, error) {
	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "failed to extract user info from context")
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	nb, err := s.store.GetNotebookByIDAndProjectID(req.Id, userInfo.ProjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "get notebook: %s", err)
		}
		return nil, status.Errorf(codes.Internal, "get notebook: %s", err)
	}

	nbProto, err := nb.V1Notebook()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "convert notebook to proto: %s", err)
	}

	switch nb.State {
	case store.NotebookStateFailed,
		store.NotebookStateInitializing,
		store.NotebookStateRunning,
		store.NotebookStateRequeued:
		return nbProto, nil
	case store.NotebookStateStopped:
	case store.NotebookStateQueued:
		if nb.QueuedAction == store.NotebookQueuedActionStart ||
			nb.QueuedAction == store.NotebookQueuedActionDelete {
			return nbProto, nil
		}
	default:
		return nil, status.Errorf(codes.Internal, "unknown notebook state: %s", nb.State)
	}

	nb, err = s.store.SetNotebookQueuedAction(nb.NotebookID, nb.Version, store.NotebookQueuedActionStart)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update notebook state: %s", err)
	}
	nbProto, err = nb.V1Notebook()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "convert notebook to proto: %s", err)
	}
	return nbProto, nil
}

// DeleteNotebook deletes a notebook.
func (s *S) DeleteNotebook(ctx context.Context, req *v1.DeleteNotebookRequest) (*v1.DeleteNotebookResponse, error) {
	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "failed to extract user info from context")
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	nb, err := s.store.GetNotebookByIDAndProjectID(req.Id, userInfo.ProjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "get notebook: %s", err)
		}
		return nil, status.Errorf(codes.Internal, "get notebook: %s", err)
	}

	if nb.QueuedAction != store.NotebookQueuedActionDelete {
		if _, err := s.store.SetNotebookQueuedAction(nb.NotebookID, nb.Version, store.NotebookQueuedActionDelete); err != nil {
			return nil, status.Errorf(codes.Internal, "update notebook state: %s", err)
		}
	}
	return &v1.DeleteNotebookResponse{}, nil
}

// ListQueuedInternalNotebooks lists queued internal notebooks.
func (ws *WS) ListQueuedInternalNotebooks(ctx context.Context, req *v1.ListQueuedInternalNotebooksRequest) (*v1.ListQueuedInternalNotebooksResponse, error) {
	clusterInfo, err := ws.extractClusterInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}

	nbs, err := ws.store.ListQueuedNotebooksByTenantIDAndClusterID(clusterInfo.TenantID, clusterInfo.ClusterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list queued notebooks: %s", err)
	}

	var inbs []*v1.InternalNotebook
	for _, nb := range nbs {
		inb, err := nb.V1InternalNotebook()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "convert notebook to proto: %s", err)
		}
		inbs = append(inbs, inb)
	}

	return &v1.ListQueuedInternalNotebooksResponse{Notebooks: inbs}, nil
}

// UpdateNotebookState updates a notebook state.
func (ws *WS) UpdateNotebookState(ctx context.Context, req *v1.UpdateNotebookStateRequest) (*v1.UpdateNotebookStateResponse, error) {
	clusterInfo, err := ws.extractClusterInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	nb, err := ws.store.GetNotebookByID(req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "notebook not found")
		}
		return nil, status.Errorf(codes.Internal, "get notebook: %s", err)
	}
	if nb.TenantID != clusterInfo.TenantID {
		return nil, status.Error(codes.NotFound, "notebook not found")
	}

	if nb.State == convertNotebookState(req.State) {
		// already in the state
		return &v1.UpdateNotebookStateResponse{}, nil
	}

	switch req.State {
	case v1.NotebookState_STATE_UNSPECIFIED:
		return nil, status.Error(codes.InvalidArgument, "state is required")
	case v1.NotebookState_INITIALIZING:
		if nb.State != store.NotebookStateQueued {
			return nil, status.Errorf(codes.FailedPrecondition, "notebook is not queued state: %s", nb.State)
		}
		if nb.QueuedAction != store.NotebookQueuedActionStart {
			return nil, status.Errorf(codes.FailedPrecondition, "notebook is not starting: %s", nb.QueuedAction)
		}
		if err := nb.MutateMessage(func(nb *v1.Notebook) {
			if nb.StartedAt == 0 {
				nb.StartedAt = time.Now().UTC().Unix()
			}
			nb.StoppedAt = 0
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "mutate message: %s", err)
		}
		if err := ws.store.SetNonQueuedStateAndMessage(nb.NotebookID, nb.Version, store.NotebookStateInitializing, nb.Message, req.Reason); err != nil {
			return nil, status.Errorf(codes.Internal, "set non queued state and message: %s", err)
		}
	case v1.NotebookState_RUNNING:
		// Rescheduler and dispatcher may update the job state at the same time, e.g. rescheduler requeues the job and
		// dispatcher updates the job state to be running. When such race condition happens, ignore the updates from dispatcher.
		if nb.State == store.NotebookStateQueued && nb.QueuedAction == store.NotebookQueuedActionRequeue {
			return &v1.UpdateNotebookStateResponse{}, nil
		}
		if nb.State != store.NotebookStateInitializing {
			return nil, status.Errorf(codes.FailedPrecondition, "notebook is not initializing state: %s", nb.State)
		}
		if err := ws.store.SetState(nb.NotebookID, nb.Version, store.NotebookStateRunning); err != nil {
			return nil, status.Errorf(codes.Internal, "set state: %s", err)
		}
	case v1.NotebookState_STOPPED:
		if nb.State != store.NotebookStateQueued {
			return nil, status.Errorf(codes.FailedPrecondition, "notebook is not queued state: %s", nb.State)
		}
		if nb.QueuedAction != store.NotebookQueuedActionStop {
			return nil, status.Errorf(codes.FailedPrecondition, "notebook is not stopping: %s", nb.QueuedAction)
		}
		if err := nb.MutateMessage(func(nb *v1.Notebook) {
			nb.StartedAt = 0
			nb.StoppedAt = time.Now().UTC().Unix()
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "mutate message: %s", err)
		}
		if err := ws.store.SetNonQueuedStateAndMessage(nb.NotebookID, nb.Version, store.NotebookStateStopped, nb.Message, req.Reason); err != nil {
			return nil, status.Errorf(codes.Internal, "set non queued state and message: %s", err)
		}
	case v1.NotebookState_DELETED:
		if nb.State != store.NotebookStateQueued {
			return nil, status.Errorf(codes.FailedPrecondition, "notebook is not queued state: %s", nb.State)
		}
		if nb.QueuedAction != store.NotebookQueuedActionDelete {
			return nil, status.Errorf(codes.FailedPrecondition, "notebook is not deleting: %s", nb.QueuedAction)
		}
		if err := nb.MutateMessage(func(nb *v1.Notebook) {
			nb.StartedAt = 0
			if nb.StoppedAt == 0 {
				nb.StoppedAt = time.Now().UTC().Unix()
			}
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "mutate message: %s", err)
		}
		if err := ws.store.SetNonQueuedStateAndMessage(nb.NotebookID, nb.Version, store.NotebookStateDeleted, nb.Message, req.Reason); err != nil {
			return nil, status.Errorf(codes.Internal, "set non queued state and message: %s", err)
		}
	case v1.NotebookState_REQUEUED:
		if nb.State != store.NotebookStateQueued {
			return nil, status.Errorf(codes.FailedPrecondition, "notebook is not queued: %s", nb.State)
		}
		if nb.QueuedAction != store.NotebookQueuedActionRequeue {
			return nil, status.Errorf(codes.FailedPrecondition, "notebook is not requeueing: %s", nb.QueuedAction)
		}
		if err := nb.MutateMessage(func(nbProto *v1.Notebook) {}); err != nil {
			return nil, status.Errorf(codes.Internal, "mutate message: %s", err)
		}
		nb.State = store.NotebookStateRequeued
		nb.Reason = ""
		if err := ws.store.UpdateNotebookForRescheduling(nb); err != nil {
			return nil, status.Errorf(codes.Internal, "update notebook: %s", err)
		}
	case v1.NotebookState_QUEUED,
		v1.NotebookState_FAILED:
		return nil, status.Errorf(codes.FailedPrecondition, "unexpected state: %s", req.State)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unknown state: %s", req.State)
	}
	return &v1.UpdateNotebookStateResponse{}, nil
}

func convertNotebookState(s v1.NotebookState) store.NotebookState {
	return store.NotebookState(strings.ToLower(s.String()))
}
