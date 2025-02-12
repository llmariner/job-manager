package cache

import (
	"fmt"
	"sync"
	"time"

	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/server/internal/store"
	"google.golang.org/protobuf/proto"
)

// Store is a cache store.
type Store struct {
	store *store.S

	// clusters is mapped from cluster ID to the clusters.
	clusters map[string]Clusters
	// mu guards all fields within this cache struct.
	mu sync.RWMutex
}

// Clusters is a map from cluster ID to clusters.
type Clusters map[string]*Cluster

// Clone returns a deep copy of the clusters.
func (c *Clusters) Clone() Clusters {
	if c == nil {
		return nil
	}
	cls := make(Clusters, len(*c))
	for k, v := range *c {
		cls[k] = v.Clone()
	}
	return cls
}

// Cluster represents a cluster.
type Cluster struct {
	ClusterID string
	UpdatedAt time.Time

	GPUNodes               []*v1.GpuNode
	ProvisionableResources []*v1.ProvisionableResource

	GPUPodsByNN map[string]*v1.GpuPod
	// AssumedGPUPodsByNN is a map from namespaced-name to the assumed GPU pods on the node.
	// This pod is bind to the node by scheduler, but not yet created.
	AssumedGPUPodsByNN map[string]*AssumedGPUPod
}

// AssumedGPUPod represents an assumed GPU pod.
type AssumedGPUPod struct {
	AllocatedCount int32
	AddedAt        time.Time
}

// Clone returns a deep copy of the cluster.
func (c *Cluster) Clone() *Cluster {
	if c == nil {
		return nil
	}
	cls := &Cluster{
		ClusterID: c.ClusterID,
		UpdatedAt: c.UpdatedAt,
		// Make a copy of the slices and maps, but the elements are not deeply copied.
		// This is fine because we don't modify the elements.
		GPUNodes:               make([]*v1.GpuNode, len(c.GPUNodes)),
		ProvisionableResources: make([]*v1.ProvisionableResource, len(c.ProvisionableResources)),
		GPUPodsByNN:            make(map[string]*v1.GpuPod, len(c.GPUPodsByNN)),
		AssumedGPUPodsByNN:     make(map[string]*AssumedGPUPod, len(c.AssumedGPUPodsByNN)),
	}
	copy(cls.GPUNodes, c.GPUNodes)
	copy(cls.ProvisionableResources, c.ProvisionableResources)
	for k, v := range c.GPUPodsByNN {
		cls.GPUPodsByNN[k] = v
	}
	for k, v := range c.AssumedGPUPodsByNN {
		cls.AssumedGPUPodsByNN[k] = v
	}
	return cls
}

// NewStore creates a new cache store.
func NewStore(store *store.S) *Store {
	return &Store{
		store:    store,
		clusters: make(map[string]Clusters),
	}
}

// ListClustersByTenantID lists clusters by tenant ID.
// If the tenant is not found in the cache, it fetches them from the store.
func (c *Store) ListClustersByTenantID(tenantID string) (map[string]*Cluster, error) {
	c.mu.RLock()
	cls, ok := c.clusters[tenantID]
	if ok {
		defer c.mu.RUnlock()
		return cls.Clone(), nil
	}
	c.mu.RUnlock()

	scls, err := c.store.ListClustersByTenantID(tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters by tenant ID: %s", err)
	}
	cls = make(map[string]*Cluster, len(scls))
	for _, scl := range scls {
		cl, err := convertToCacheCluster(scl)
		if err != nil {
			return nil, fmt.Errorf("failed to convert to cache cluster: %s", err)
		}
		cls[scl.ClusterID] = cl
	}
	c.mu.Lock()
	c.clusters[tenantID] = cls
	c.mu.Unlock()
	return cls, nil
}

// assumedPodExpiration is the expiration time for assumed pods.
// If the pod is not added within this time, it is removed from the assumed
// pod map in the next cluster update.
const assumedPodExpiration = time.Minute

// AddOrUpdateCluster adds or updates a cluster.
// If the tenant is not found in the cache, it fetches them from the store.
func (c *Store) AddOrUpdateCluster(cluster *store.Cluster) error {
	cl, err := convertToCacheCluster(cluster)
	if err != nil {
		return fmt.Errorf("failed to convert to cache cluster: %s", err)
	}

	oldCl, ok, err := c.getCluster(cluster.TenantID, cluster.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %s", err)
	}
	if ok {
		// TODO(aya): rethink the better expiration logic.
		for nn, pod := range oldCl.AssumedGPUPodsByNN {
			if _, ok := cl.GPUPodsByNN[nn]; !ok && time.Since(pod.AddedAt) < assumedPodExpiration {
				cl.AssumedGPUPodsByNN[nn] = pod
			}
		}
	}

	c.mu.Lock()
	c.clusters[cluster.TenantID][cl.ClusterID] = cl
	c.mu.Unlock()
	return nil
}

// AddAssumedPod adds an assumed pod to the cache.
// If the tenant is not found in the cache, it fetches them from the store.
func (c *Store) AddAssumedPod(tenantID, clusterID string, pod *v1.GpuPod) error {
	cls, ok, err := c.getCluster(tenantID, clusterID)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %s", err)
	}
	if !ok {
		return fmt.Errorf("cluster not found: %s", clusterID)
	}
	cls.AssumedGPUPodsByNN[pod.NamespacedName] = &AssumedGPUPod{
		AllocatedCount: pod.AllocatedCount,
		AddedAt:        time.Now(),
	}

	c.mu.Lock()
	c.clusters[tenantID][clusterID] = cls
	c.mu.Unlock()
	return nil
}

func (c *Store) getCluster(tenantID, clusterID string) (*Cluster, bool, error) {
	c.mu.RLock()
	cls, ok := c.clusters[tenantID]
	if ok {
		defer c.mu.RUnlock()
		cl, ok := cls[clusterID]
		return cl.Clone(), ok, nil
	}
	c.mu.RUnlock()
	cls, err := c.ListClustersByTenantID(tenantID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to list clusters by tenant ID: %s", err)
	}
	cl, ok := cls[clusterID]
	return cl, ok, nil
}

func convertToCacheCluster(c *store.Cluster) (*Cluster, error) {
	var status v1.ClusterStatus
	if err := proto.Unmarshal(c.Status, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cluster status: %s", err)
	}
	gpuPodsByNN := map[string]*v1.GpuPod{}
	for _, pod := range status.GpuPods {
		gpuPodsByNN[pod.NamespacedName] = pod
	}
	return &Cluster{
		ClusterID:              c.ClusterID,
		UpdatedAt:              c.UpdatedAt,
		GPUNodes:               status.GpuNodes,
		ProvisionableResources: status.ProvisionableResources,
		GPUPodsByNN:            gpuPodsByNN,
		AssumedGPUPodsByNN:     make(map[string]*AssumedGPUPod),
	}, nil
}
