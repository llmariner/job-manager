package server

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	"github.com/llmariner/api-usage/pkg/sender"
	fv1 "github.com/llmariner/file-manager/api/v1"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/server/internal/config"
	"github.com/llmariner/job-manager/server/internal/k8s"
	"github.com/llmariner/job-manager/server/internal/scheduler"
	"github.com/llmariner/job-manager/server/internal/store"
	mv1 "github.com/llmariner/model-manager/api/v1"
	"github.com/llmariner/rbac-manager/pkg/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
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

type schedulerI interface {
	Schedule(userInfo *auth.UserInfo) (scheduler.SchedulingResult, error)
}

// New creates a server.
func New(
	store *store.S,
	fileGetClient fileGetClient,
	modelClient modelClient,
	k8sClientFactory k8s.ClientFactory,
	scheduler schedulerI,
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
		scheduler:        scheduler,
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
	v1.UnimplementedJobServiceServer

	srv *grpc.Server

	store            *store.S
	fileGetClient    fileGetClient
	modelClient      modelClient
	k8sClientFactory k8s.ClientFactory
	scheduler        schedulerI

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
	// TODO(kenji): Change this to the admin service if we decide not to expose this to end users.
	v1.RegisterJobServiceServer(grpcServer, s)
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

// ListClusters lists clusters.
func (s *S) ListClusters(ctx context.Context, req *v1.ListClustersRequest) (*v1.ListClustersResponse, error) {
	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "failed to extract user info from context")
	}
	accessibleClusters := map[string]bool{}
	for _, env := range userInfo.AssignedKubernetesEnvs {
		accessibleClusters[env.ClusterID] = true
	}

	clusters, err := s.store.ListClustersByTenantID(userInfo.TenantID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list clusters: %s", err)
	}

	// Filter out clusters that the user does not have access.
	var cs []*v1.Cluster
	for _, c := range clusters {
		if !accessibleClusters[c.ClusterID] {
			continue
		}

		var st v1.ClusterStatus
		if err := proto.Unmarshal(c.Status, &st); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal cluster status: %s", err)
		}
		cs = append(cs, &v1.Cluster{
			Id:     c.ClusterID,
			Status: &st,
		})
	}
	sort.Slice(cs, func(i, j int) bool {
		return cs[i].Id < cs[j].Id
	})

	return &v1.ListClustersResponse{
		Clusters: cs,
	}, nil
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
