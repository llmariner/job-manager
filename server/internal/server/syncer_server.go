package server

import (
	"context"
	"fmt"
	"net"
	"sort"

	"github.com/go-logr/logr"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/server/internal/config"
	"github.com/llmariner/job-manager/server/internal/k8s"
	"github.com/llmariner/rbac-manager/pkg/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewSyncerServiceServer creates a new syncer service server.
func NewSyncerServiceServer(logger logr.Logger, k8sClientFactory k8s.ClientFactory, scheduler schedulerI, cache cacheI) *SS {
	return &SS{
		logger:           logger.WithName("syncer"),
		k8sClientFactory: k8sClientFactory,
		scheduler:        scheduler,
		cache:            cache,
	}
}

// SS is a server for syncer services.
type SS struct {
	v1.UnimplementedSyncerServiceServer

	srv              *grpc.Server
	k8sClientFactory k8s.ClientFactory
	scheduler        schedulerI
	cache            cacheI
	logger           logr.Logger
}

// Run runs the syncer service server.
func (ss *SS) Run(ctx context.Context, port int, authConfig config.AuthConfig) error {
	ss.logger.Info("Starting syncer service server...", "port", port)

	var opt grpc.ServerOption
	if authConfig.Enable {
		ai, err := auth.NewInterceptor(ctx, auth.Config{
			RBACServerAddr: authConfig.RBACInternalServerAddr,
			GetAccessResourceForGRPCRequest: func(fullMethod string) string {
				if fullMethod == "/llmariner.syncer.server.v1.SyncerService/ListClusterIDs" {
					return "api.clusters"
				}
				return "api.k8s.namespaced"
			},
		})
		if err != nil {
			return err
		}
		opt = grpc.ChainUnaryInterceptor(ai.Unary())
	} else {
		opt = grpc.ChainUnaryInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
			return handler(fakeAuthInto(ctx), req)
		})
	}

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
	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "name is required")
	}
	if req.Version == "" {
		return nil, status.Errorf(codes.InvalidArgument, "version is required")
	}
	if req.Resource == "" {
		return nil, status.Errorf(codes.InvalidArgument, "resource is required")
	}
	if len(req.Data) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "data is required")
	}

	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to extract user info from context")
	}
	apikey, err := auth.ExtractTokenFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// TODO(aya): Schedule to the cluster where it was created If the resource is not newly created.
	// TODO(kenji): Fix the gpu count.
	var gpuCount int
	if r := req.Resources; r != nil {
		gpuCount = int(r.GpuLimit)
	}
	sresult, err := ss.scheduler.Schedule(userInfo, "", gpuCount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "schedule: %s", err)
	}
	if err := ss.cache.AddAssumedPod(userInfo.TenantID, sresult.ClusterID, &v1.GpuPod{
		AllocatedCount: int32(gpuCount),
		NamespacedName: fmt.Sprintf("%s/%s", sresult.Namespace, req.Name),
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "add assumed pod: %s", err)
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
	if req.ClusterId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "cluster ID is required")
	}
	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "name is required")
	}
	if req.Version == "" {
		return nil, status.Errorf(codes.InvalidArgument, "version is required")
	}
	if req.Resource == "" {
		return nil, status.Errorf(codes.InvalidArgument, "resource is required")
	}

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

// ListClusterIDs lists cluster IDs.
func (ss *SS) ListClusterIDs(ctx context.Context, req *v1.ListClusterIDsRequest) (*v1.ListClusterIDsResponse, error) {
	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "failed to extract user info from context")
	}
	accessibleClusters := map[string]bool{}
	for _, env := range userInfo.AssignedKubernetesEnvs {
		accessibleClusters[env.ClusterID] = true
	}

	resp := &v1.ListClusterIDsResponse{}
	for id := range accessibleClusters {
		resp.Ids = append(resp.Ids, id)
	}
	sort.Slice(resp.Ids, func(i, j int) bool {
		return resp.Ids[i] < resp.Ids[j]
	})
	return resp, nil
}
