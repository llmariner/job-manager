package server

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// CreateNotebook creates a notebook.
func (s *S) CreateNotebook(ctx context.Context, req *v1.CreateNotebookRequest) (*v1.Notebook, error) {
	userInfo, err := s.extractUserInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
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

	// TODO(aya): validate resources

	nbID := newNotebookID()
	nbProto := &v1.Notebook{
		Id:        nbID,
		Name:      req.Name,
		CreatedAt: time.Now().UTC().Unix(),
		Image:     image,
		Resources: req.Resources,
		Envs:      req.Envs,
		Status:    string(store.NotebookStateQueued),
	}
	msg, err := proto.Marshal(nbProto)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal notebook: %s", err)
	}

	nb := &store.Notebook{
		NotebookID:          nbID,
		Image:               image,
		Message:             msg,
		State:               store.NotebookStateQueued,
		TenantID:            fakeTenantID,
		OrganizationID:      userInfo.OrganizationID,
		ProjectID:           userInfo.ProjectID,
		KubernetesNamespace: userInfo.KubernetesNamespace,
	}
	if err := s.store.CreateNotebook(nb); err != nil {
		return nil, status.Errorf(codes.Internal, "create notebook: %s", err)
	}
	return nbProto, nil
}

// ListNotebooks lists notebooks.
func (s *S) ListNotebooks(ctx context.Context, req *v1.ListNotebooksRequest) (*v1.ListNotebooksResponse, error) {
	userInfo, err := s.extractUserInfoFromContext(ctx)
	if err != nil {
		return nil, err
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

	nbs, hasMore, err := s.store.ListNotebooksByProjectIDWithPagination(userInfo.ProjectID, afterID, int(limit))
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
	return &v1.ListNotebooksResponse{
		Notebooks: nbProtos,
		HasMore:   hasMore,
	}, nil
}

// GetNotebook gets a notebook.
func (s *S) GetNotebook(ctx context.Context, req *v1.GetNotebookRequest) (*v1.Notebook, error) {
	userInfo, err := s.extractUserInfoFromContext(ctx)
	if err != nil {
		return nil, err
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

func newNotebookID() string {
	return uuid.New().String()
}
