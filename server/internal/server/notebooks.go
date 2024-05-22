package server

import (
	"context"
	"time"

	"github.com/google/uuid"
	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
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
		image = t
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

func newNotebookID() string {
	return uuid.New().String()
}
