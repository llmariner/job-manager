package server

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr/testr"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/server/internal/store"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
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

func TestUpdateClusterStatus(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	_, err := st.GetClusterByID(defaultClusterID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))

	srv := NewWorkerServiceServer(st, testr.New(t))
	req := &v1.UpdateClusterStatusRequest{
		ClusterStatus: &v1.ClusterStatus{},
	}
	_, err = srv.UpdateClusterStatus(fakeAuthInto(context.Background()), req)
	assert.NoError(t, err)

	got, err := st.GetClusterByID(defaultClusterID)
	assert.NoError(t, err)
	assert.Equal(t, defaultClusterID, got.ClusterID)

}
