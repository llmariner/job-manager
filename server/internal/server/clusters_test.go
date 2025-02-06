package server

import (
	"context"
	"testing"

	"github.com/go-logr/logr/testr"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/server/internal/store"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestListClusters(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	status := &v1.ClusterStatus{
		GpuNodes: []*v1.GpuNode{
			{
				ResourceName:     "nvidia.com/gpu",
				AllocatableCount: 1,
			},
		},
	}
	stb, err := proto.Marshal(status)
	assert.NoError(t, err)

	cs := []*store.Cluster{
		{
			ClusterID: defaultClusterID,
			TenantID:  defaultTenantID,
			Status:    stb,
		},
		{
			ClusterID: "different-cluster",
			TenantID:  defaultTenantID,
			Status:    stb,
		},
		{
			ClusterID: "different-cluster",
			TenantID:  "different-tenant",
			Status:    stb,
		},
	}
	for _, c := range cs {
		err := st.CreateOrUpdateCluster(c)
		assert.NoError(t, err)
	}

	srv := New(st, nil, nil, nil, nil, nil, nil, testr.New(t))

	ctx := fakeAuthInto(context.Background())
	resp, err := srv.ListClusters(ctx, &v1.ListClustersRequest{})
	assert.NoError(t, err)
	assert.Len(t, resp.Clusters, 1)
	c := resp.Clusters[0]
	assert.Equal(t, defaultClusterID, c.Id)
	assert.True(t, proto.Equal(status, c.Status))
}
