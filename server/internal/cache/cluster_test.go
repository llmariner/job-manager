package cache

import (
	"testing"
	"time"

	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/server/internal/store"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestCluster_Clone(t *testing.T) {
	cls := &Cluster{
		ClusterID: "test-cluster",
		UpdatedAt: time.Now(),
		GPUNodes: []*v1.GpuNode{
			{ResourceName: "r0", AllocatableCount: 2},
			{ResourceName: "r2", AllocatableCount: 1},
		},
		ProvisionableResources: []*v1.ProvisionableResource{
			{InstanceFamily: "f0", InstanceType: "t0"},
		},
		GPUPodsByNN: map[string]*v1.GpuPod{
			"pod-1": {
				ResourceName:   "r0",
				AllocatedCount: 1,
				NamespacedName: "ns-1/pod-1",
			},
		},
		AssumedGPUPodsByNN: map[string]*AssumedGPUPod{
			"pod-2": {
				AllocatedCount: 1,
				AddedAt:        time.Now(),
			},
		},
	}
	gotCls := cls.Clone()
	assert.Equal(t, cls, gotCls)
	assert.Same(t, cls.GPUNodes[0], gotCls.GPUNodes[0])
	assert.NotSame(t, cls.GPUNodes, gotCls.GPUNodes)
	assert.NotSame(t, cls.GPUPodsByNN, gotCls.GPUPodsByNN)
	assert.NotSame(t, cls.AssumedGPUPodsByNN, gotCls.AssumedGPUPodsByNN)
}

func TestCache(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	clusters := []*store.Cluster{
		stCluster(t, "t0", "c0", "ns-1/pod-1"),
		stCluster(t, "t0", "c1", "ns-1/pod-2", "ns-1/pod-3"),
		stCluster(t, "t1", "c2"),
	}
	for _, c := range clusters {
		err := st.CreateOrUpdateCluster(c)
		assert.NoError(t, err)
	}

	c := NewStore(st)
	// list from store
	gotT0Cls, err := c.ListClustersByTenantID("t0")
	assert.NoError(t, err)
	assert.Len(t, gotT0Cls, 2)
	// list from cache
	gotT0Cls2, err := c.ListClustersByTenantID("t0")
	assert.NoError(t, err)
	assert.Len(t, gotT0Cls2, 2)
	assert.Equal(t, gotT0Cls, gotT0Cls2)
	assert.NotSame(t, gotT0Cls, gotT0Cls2)

	gotT1Cls, err := c.ListClustersByTenantID("t1")
	assert.NoError(t, err)
	assert.Len(t, gotT1Cls, 1)
	gotEmpty, err := c.ListClustersByTenantID("unknown")
	assert.NoError(t, err)
	assert.Len(t, gotEmpty, 0)

	// add assumed pod to cache
	assumedPod := &v1.GpuPod{ResourceName: "r0", AllocatedCount: 1, NamespacedName: "ns-1/pod-4"}
	err = c.AddAssumedPod("t0", "c0", assumedPod)
	assert.NoError(t, err)
	assumedPod2 := &v1.GpuPod{ResourceName: "r0", AllocatedCount: 1, NamespacedName: "ns-1/pod-5"}
	err = c.AddAssumedPod("t0", "c0", assumedPod2)
	assert.NoError(t, err)
	gotT0Cls3, err := c.ListClustersByTenantID("t0")
	assert.NoError(t, err)
	assert.Len(t, gotT0Cls3["c0"].GPUPodsByNN, 1)
	assert.Len(t, gotT0Cls3["c0"].AssumedGPUPodsByNN, 2)

	err = c.AddAssumedPod("t0", "unknown", assumedPod)
	assert.ErrorContains(t, err, "cluster not found: unknown")
	err = c.AddAssumedPod("unknown", "unknown", assumedPod)
	assert.ErrorContains(t, err, "cluster not found: unknown")

	// update cluster c0
	newT0Cl0 := stCluster(t, "t0", "c0", "ns-1/pod-1", "ns-1/pod-4")
	err = c.AddOrUpdateCluster(newT0Cl0)
	assert.NoError(t, err)
	gotT0Cls4, err := c.ListClustersByTenantID("t0")
	assert.NoError(t, err)
	assert.Len(t, gotT0Cls4["c0"].GPUPodsByNN, 2)
	assert.Len(t, gotT0Cls4["c0"].AssumedGPUPodsByNN, 1)
	// add cluster c3
	err = c.AddOrUpdateCluster(stCluster(t, "t0", "c3"))
	assert.NoError(t, err)
}

func stCluster(t *testing.T, tid, cid string, pods ...string) *store.Cluster {
	status := &v1.ClusterStatus{
		GpuNodes: []*v1.GpuNode{
			{ResourceName: "r0", AllocatableCount: 3},
			{ResourceName: "r1", AllocatableCount: 1},
			{ResourceName: "r0", AllocatableCount: 2},
		},
		ProvisionableResources: []*v1.ProvisionableResource{
			{InstanceFamily: "f0", InstanceType: "t0"},
			{InstanceFamily: "f1", InstanceType: "t1"},
		},
		GpuPods: []*v1.GpuPod{},
	}
	for _, p := range pods {
		status.GpuPods = append(status.GpuPods,
			&v1.GpuPod{ResourceName: "r0", AllocatedCount: 1, NamespacedName: p})
	}
	b, err := proto.Marshal(status)
	assert.NoError(t, err)

	return &store.Cluster{
		ClusterID: cid,
		Name:      cid,
		TenantID:  tid,
		Status:    b,
	}
}
