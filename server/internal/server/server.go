package server

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/go-logr/logr"
	"github.com/llmariner/api-usage/pkg/sender"
	fv1 "github.com/llmariner/file-manager/api/v1"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/server/internal/config"
	"github.com/llmariner/job-manager/server/internal/k8s"
	"github.com/llmariner/job-manager/server/internal/store"
	mv1 "github.com/llmariner/model-manager/api/v1"
	"github.com/llmariner/rbac-manager/pkg/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
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
	logger logr.Logger,
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
		logger:           logger.WithName("grpc"),
	}
}

// S is a server.
type S struct {
	v1.UnimplementedFineTuningServiceServer
	v1.UnimplementedWorkspaceServiceServer
	v1.UnimplementedBatchServiceServer

	srv *grpc.Server

	store            *store.S
	fileGetClient    fileGetClient
	modelClient      modelClient
	k8sClientFactory k8s.ClientFactory

	nbImageTypes   map[string]string
	nbImageTypeStr string

	batchJobImages map[string]string

	logger logr.Logger
}

// Run starts the gRPC server.
func (s *S) Run(ctx context.Context, port int, authConfig config.AuthConfig, usage sender.UsageSetter) error {
	s.logger.Info("Starting gRPC server", "port", port)

	var opt grpc.ServerOption
	if authConfig.Enable {
		ai, err := auth.NewInterceptor(ctx, auth.Config{
			RBACServerAddr: authConfig.RBACInternalServerAddr,
			GetAccessResourceForGRPCRequest: func(fullMethod string) string {
				switch {
				case strings.HasPrefix(fullMethod, "/llmoperator.workspace."),
					strings.HasPrefix(fullMethod, "/llmariner.workspace."):
					return "api.workspaces.notebooks"
				}
				return "api.fine_tuning.jobs"
			},
		})
		if err != nil {
			return err
		}
		opt = grpc.ChainUnaryInterceptor(ai.Unary("/grpc.health.v1.Health/Check"), sender.Unary(usage))
	} else {
		fakeAuth := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
			if info.FullMethod == "/grpc.health.v1.Health/Check" {
				// Skip authentication for health check
				return handler(ctx, req)
			}
			return handler(fakeAuthInto(ctx), req)
		}
		opt = grpc.ChainUnaryInterceptor(fakeAuth, sender.Unary(usage))
	}

	grpcServer := grpc.NewServer(opt)
	v1.RegisterFineTuningServiceServer(grpcServer, s)
	v1.RegisterWorkspaceServiceServer(grpcServer, s)
	v1.RegisterBatchServiceServer(grpcServer, s)
	reflection.Register(grpcServer)

	healthCheck := health.NewServer()
	healthCheck.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthCheck)

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

// fakeAuthInto sets dummy user info and token into the context.
func fakeAuthInto(ctx context.Context) context.Context {
	// Set dummy user info and token
	ctx = auth.AppendUserInfoToContext(ctx, auth.UserInfo{
		OrganizationID: "default",
		ProjectID:      defaultProjectID,
		AssignedKubernetesEnvs: []auth.AssignedKubernetesEnv{
			{
				ClusterID: defaultClusterID,
				Namespace: "default",
			},
		},
		TenantID: defaultTenantID,
	})
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("authorization", "Bearer token"))
	return ctx
}
