package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	fv1 "github.com/llm-operator/file-manager/api/v1"
	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	"github.com/llm-operator/job-manager/server/internal/config"
	mv1 "github.com/llm-operator/model-manager/api/v1"
	"github.com/llm-operator/rbac-manager/pkg/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

const (
	defaultProjectID = "default"

	defaultPageSize = 20
	maxPageSize     = 100
)

type fileGetClient interface {
	GetFile(ctx context.Context, in *fv1.GetFileRequest, opts ...grpc.CallOption) (*fv1.File, error)
}

type modelClient interface {
	GetModel(ctx context.Context, in *mv1.GetModelRequest, opts ...grpc.CallOption) (*mv1.Model, error)
}

type k8sJobClient interface {
	CancelJob(ctx context.Context, job *v1.Job, namespace string) error
}

// New creates a server.
func New(
	store *store.S,
	fileGetClient fileGetClient,
	modelClient modelClient,
	k8sJobClient k8sJobClient,
) *S {
	return &S{
		store:         store,
		fileGetClient: fileGetClient,
		modelClient:   modelClient,
		k8sJobClient:  k8sJobClient,
	}
}

// S is a server.
type S struct {
	v1.UnimplementedFineTuningServiceServer
	v1.UnimplementedWorkspaceServiceServer

	srv *grpc.Server

	enableAuth bool

	store         *store.S
	fileGetClient fileGetClient
	modelClient   modelClient
	k8sJobClient  k8sJobClient
}

// Run starts the gRPC server.
func (s *S) Run(ctx context.Context, port int, authConfig config.AuthConfig) error {
	log.Printf("Starting server on port %d\n", port)

	var opts []grpc.ServerOption
	if authConfig.Enable {
		ai, err := auth.NewInterceptor(ctx, auth.Config{
			RBACServerAddr: authConfig.RBACInternalServerAddr,
			GetAccessResourceForGRPCRequest: func(fullMethod string) string {
				switch {
				case strings.HasPrefix(fullMethod, "/llmoperator.workspace."):
					return "api.workspaces.notebooks"
				}
				return "api.fine_tuning.jobs"
			},
		})
		if err != nil {
			return err
		}
		opts = append(opts, grpc.ChainUnaryInterceptor(ai.Unary()))
		s.enableAuth = true
	}

	grpcServer := grpc.NewServer(opts...)
	v1.RegisterFineTuningServiceServer(grpcServer, s)
	v1.RegisterWorkspaceServiceServer(grpcServer, s)
	reflection.Register(grpcServer)

	s.srv = grpcServer

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("listen: %s", err)
	}
	if err := grpcServer.Serve(l); err != nil {
		return fmt.Errorf("serve: %s", err)
	}
	return nil
}

// Stop stops the gRPC server.
func (s *S) Stop() {
	s.srv.Stop()
}

func (s *S) extractUserInfoFromContext(ctx context.Context) (*auth.UserInfo, error) {
	if !s.enableAuth {
		return &auth.UserInfo{
			OrganizationID:      "default",
			ProjectID:           defaultProjectID,
			KubernetesNamespace: "default",
		}, nil
	}
	var ok bool
	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user info not found")
	}
	return userInfo, nil
}
