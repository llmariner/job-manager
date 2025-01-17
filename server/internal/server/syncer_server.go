package server

import (
	"context"
	"fmt"
	"net"

	"github.com/go-logr/logr"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/server/internal/k8s"
	"github.com/llmariner/rbac-manager/pkg/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewSyncerServiceServer creates a new syncer service server.
func NewSyncerServiceServer(logger logr.Logger, k8sClientFactory k8s.ClientFactory, scheduler schedulerI) *SS {
	return &SS{
		logger:           logger.WithName("syncer"),
		k8sClientFactory: k8sClientFactory,
		scheduler:        scheduler,
	}
}

// SS is a server for syncer services.
type SS struct {
	v1.UnimplementedSyncerServiceServer

	srv              *grpc.Server
	k8sClientFactory k8s.ClientFactory
	scheduler        schedulerI
	logger           logr.Logger
}

// Run runs the syncer service server.
func (ss *SS) Run(ctx context.Context, port int) error {
	ss.logger.Info("Starting syncer service server...", "port", port)

	// TODO: support auth
	fakeAuth := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		return handler(fakeAuthInto(ctx), req)
	}
	opt := grpc.ChainUnaryInterceptor(fakeAuth)

	srv := grpc.NewServer(opt)
	v1.RegisterSyncerServiceServer(srv, ss)
	reflection.Register(srv)
	ss.srv = srv

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	if err := srv.Serve(l); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

// PatchKubernetesObject applies a kubernetes object.
func (ss *SS) PatchKubernetesObject(ctx context.Context, req *v1.PatchKubernetesObjectRequest) (*v1.PatchKubernetesObjectResponse, error) {
	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to extract user info from context")
	}
	apikey, err := auth.ExtractTokenFromContext(ctx)
	if err != nil {
		return nil, err
	}

	sresult, err := ss.scheduler.Schedule(userInfo)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "schedule: %s", err)
	}
	clusterID := sresult.ClusterID

	if sresult.Namespace != req.Namespace {
		// TODO(aya): rethink the namespace handling
		return nil, status.Errorf(codes.NotFound, "not found the namespace")
	}

	client, err := ss.k8sClientFactory.NewDynamicClient(clusterID, apikey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create k8s client: %s", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    req.Group,
		Version:  req.Version,
		Resource: req.Resource,
	}
	uobj, err := client.PatchResource(ctx, req.Name, req.Namespace, gvr, req.Data)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "patch k8s object: %s", err)
	}
	return &v1.PatchKubernetesObjectResponse{
		ClusterId: clusterID,
		Uid:       string(uobj.GetUID()),
	}, nil
}

// DeleteKubernetesObject deletes a kubernetes object.
func (ss *SS) DeleteKubernetesObject(ctx context.Context, req *v1.DeleteKubernetesObjectRequest) (*v1.DeleteKubernetesObjectResponse, error) {
	apikey, err := auth.ExtractTokenFromContext(ctx)
	if err != nil {
		return nil, err
	}

	client, err := ss.k8sClientFactory.NewDynamicClient(req.ClusterId, apikey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create k8s client: %s", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    req.Group,
		Version:  req.Version,
		Resource: req.Resource,
	}
	if err := client.DeleteResource(ctx, req.Name, req.Namespace, gvr); err != nil {
		return nil, status.Errorf(codes.Internal, "delete k8s object: %s", err)
	}
	return &v1.DeleteKubernetesObjectResponse{}, nil
}
