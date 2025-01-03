package server

import (
	"context"
	"fmt"
	"net"

	"github.com/go-logr/logr"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/server/internal/config"
	"github.com/llmariner/job-manager/server/internal/store"
	"github.com/llmariner/rbac-manager/pkg/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const (
	defaultClusterID = "default"
)

// NewWorkerServiceServer creates a new worker service server.
func NewWorkerServiceServer(s *store.S, logger logr.Logger) *WS {
	return &WS{
		store:  s,
		logger: logger.WithName("worker"),
	}
}

// WS is a server for worker services.
type WS struct {
	v1.UnimplementedFineTuningWorkerServiceServer
	v1.UnimplementedWorkspaceWorkerServiceServer
	v1.UnimplementedBatchWorkerServiceServer
	v1.UnimplementedJobWorkerServiceServer

	srv    *grpc.Server
	store  *store.S
	logger logr.Logger

	enableAuth bool
}

// Run runs the worker service server.
func (ws *WS) Run(ctx context.Context, port int, authConfig config.AuthConfig) error {
	ws.logger.Info("Starting worker service server...", "port", port)

	var opts []grpc.ServerOption
	if authConfig.Enable {
		ai, err := auth.NewWorkerInterceptor(ctx, auth.WorkerConfig{
			RBACServerAddr: authConfig.RBACInternalServerAddr,
		})
		if err != nil {
			return err
		}
		opts = append(opts, grpc.ChainUnaryInterceptor(ai.Unary()))
		ws.enableAuth = true
	}

	srv := grpc.NewServer(opts...)

	v1.RegisterFineTuningWorkerServiceServer(srv, ws)
	v1.RegisterWorkspaceWorkerServiceServer(srv, ws)
	v1.RegisterBatchWorkerServiceServer(srv, ws)
	v1.RegisterJobWorkerServiceServer(srv, ws)
	reflection.Register(srv)

	ws.srv = srv

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	if err := srv.Serve(l); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

// Stop stops the worker service server.
func (ws *WS) Stop() {
	ws.srv.Stop()
}

func (ws *WS) extractClusterInfoFromContext(ctx context.Context) (*auth.ClusterInfo, error) {
	if !ws.enableAuth {
		return &auth.ClusterInfo{
			ClusterID: defaultClusterID,
			TenantID:  defaultTenantID,
		}, nil
	}
	clusterInfo, ok := auth.ExtractClusterInfoFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "cluster info not found")
	}
	return clusterInfo, nil
}

// UpdateClusterStatus updates the cluster status.
func (ws *WS) UpdateClusterStatus(
	ctx context.Context,
	req *v1.UpdateClusterStatusRequest,
) (*v1.UpdateClusterStatusResponse, error) {
	clusterInfo, err := ws.extractClusterInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.ClusterStatus == nil {
		return nil, status.Error(codes.InvalidArgument, "cluster_status is required")
	}

	b, err := proto.Marshal(req.ClusterStatus)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal proto: %s", err)
	}

	c := &store.Cluster{
		ClusterID: clusterInfo.ClusterID,
		TenantID:  clusterInfo.TenantID,
		Status:    b,
	}

	if err := ws.store.CreateOrUpdateCluster(c); err != nil {
		return nil, status.Errorf(codes.Internal, "create or update cluster: %s", err)
	}

	return &v1.UpdateClusterStatusResponse{}, nil
}
