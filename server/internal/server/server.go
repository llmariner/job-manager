package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	fv1 "github.com/llm-operator/file-manager/api/v1"
	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/server/internal/config"
	"github.com/llm-operator/job-manager/server/internal/k8s"
	"github.com/llm-operator/job-manager/server/internal/store"
	mv1 "github.com/llm-operator/model-manager/api/v1"
	"github.com/llm-operator/rbac-manager/pkg/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

const (
	defaultProjectID = "default"
	defaultTenantID  = "default-tenant-id"

	defaultPageSize = 20
	maxPageSize     = 100
)

type fileGetClient interface {
	GetFile(ctx context.Context, in *fv1.GetFileRequest, opts ...grpc.CallOption) (*fv1.File, error)
}

type modelClient interface {
	GetModel(ctx context.Context, in *mv1.GetModelRequest, opts ...grpc.CallOption) (*mv1.Model, error)
}

// New creates a server.
func New(
	store *store.S,
	fileGetClient fileGetClient,
	modelClient modelClient,
	k8sClientFactory k8s.ClientFactory,
	nbImageTypes map[string]string,
	batchJobImages map[string]string,
) *S {
	nbtypes := make([]string, 0, len(nbImageTypes))
	for t := range nbImageTypes {
		nbtypes = append(nbtypes, t)
	}
	return &S{
		store:            store,
		fileGetClient:    fileGetClient,
		modelClient:      modelClient,
		k8sClientFactory: k8sClientFactory,
		nbImageTypes:     nbImageTypes,
		nbImageTypeStr:   strings.Join(nbtypes, ", "),
		batchJobImages:   batchJobImages,
	}
}

// S is a server.
type S struct {
	v1.UnimplementedFineTuningServiceServer
	v1.UnimplementedWorkspaceServiceServer
	v1.UnimplementedBatchServiceServer

	srv *grpc.Server

	enableAuth bool

	store            *store.S
	fileGetClient    fileGetClient
	modelClient      modelClient
	k8sClientFactory k8s.ClientFactory

	nbImageTypes   map[string]string
	nbImageTypeStr string

	batchJobImages map[string]string
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
	v1.RegisterBatchServiceServer(grpcServer, s)
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
			OrganizationID: "default",
			ProjectID:      defaultProjectID,
			AssignedKubernetesEnvs: []auth.AssignedKubernetesEnv{
				{
					ClusterID: defaultClusterID,
					Namespace: "default",
				},
			},
			TenantID: defaultTenantID,
		}, nil
	}
	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user info not found")
	}
	return userInfo, nil
}

func (s *S) extractTokenFromContext(ctx context.Context) (string, error) {
	if !s.enableAuth {
		return "token", nil
	}
	token, err := auth.ExtractTokenFromContext(ctx)
	if err != nil {
		return "", status.Errorf(codes.Internal, "extract token: %s", err)
	}
	return token, nil
}
