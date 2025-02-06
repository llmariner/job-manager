package server

import (
	"context"
	"sort"

	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/rbac-manager/pkg/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

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
			Id:            c.ClusterID,
			Status:        &st,
			LastUpdatedAt: c.UpdatedAt.UnixNano(),
		})
	}
	sort.Slice(cs, func(i, j int) bool {
		return cs[i].Id < cs[j].Id
	})

	return &v1.ListClustersResponse{
		Clusters: cs,
	}, nil
}
