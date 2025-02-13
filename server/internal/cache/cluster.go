package cache

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
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

	logger logr.Logger
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
	ClusterID   string
	ClusterName string
	UpdatedAt   time.Time

	GPUNodes               []*v1.GpuNode
	ProvisionableResources []*v1.ProvisionableResource

	GPUPods []*v1.GpuPod
	// AssumedGPUPodsByKey is a map from key to the assumed GPU pods on the node.
	// This pod is bound to the node by scheduler, but not yet created.
	AssumedGPUPodsByKey map[string]*AssumedGPUPod
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
		ClusterID:   c.ClusterID,
		ClusterName: c.ClusterName,
		UpdatedAt:   c.UpdatedAt,
		// Make a copy of the slices and maps, but the elements are not deeply copied.
		// This is fine because we don't modify the elements.
		GPUNodes:               make([]*v1.GpuNode, len(c.GPUNodes)),
		ProvisionableResources: make([]*v1.ProvisionableResource, len(c.ProvisionableResources)),
		GPUPods:                make([]*v1.GpuPod, len(c.GPUPods)),
		AssumedGPUPodsByKey:    make(map[string]*AssumedGPUPod, len(c.AssumedGPUPodsByKey)),
	}
	copy(cls.GPUNodes, c.GPUNodes)
	copy(cls.ProvisionableResources, c.ProvisionableResources)
	copy(cls.GPUPods, c.GPUPods)
	for k, v := range c.AssumedGPUPodsByKey {
		cls.AssumedGPUPodsByKey[k] = v
	}
	return cls
}

// NewStore creates a new cache store.
func NewStore(store *store.S, log logr.Logger) *Store {
	return &Store{
		store:    store,
		clusters: make(map[string]Clusters),
		logger:   log,
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
		// NOTE: If the pod is quickly completed, it would not be recorded in GpuPods,
		// it remains in the assumed pod map until the expiration time.
		for key, pod := range oldCl.AssumedGPUPodsByKey {
			if time.Since(pod.AddedAt) < assumedPodExpiration &&
				!nnHasPrefix(cl.GPUPods, key) {
				cl.AssumedGPUPodsByKey[key] = pod
			}
		}
		c.logger.V(3).Info("updated cluster", "ID", cl.ClusterID, "gpuPods", cl.GPUPods, "assumedPods", cl.AssumedGPUPodsByKey)
	}

	c.mu.Lock()
	c.clusters[cluster.TenantID][cl.ClusterID] = cl
	c.mu.Unlock()
	return nil
}

func nnHasPrefix(pods []*v1.GpuPod, key string) bool {
	for _, p := range pods {
		if strings.HasPrefix(p.NamespacedName, key) {
			return true
		}
	}
	return false
}

// AddAssumedPod adds an assumed pod to the cache.
// If the tenant is not found in the cache, it fetches them from the store.
func (c *Store) AddAssumedPod(tenantID, clusterID, key string, gpuCount int) error {
	if gpuCount == 0 {
		// ignore non GPU pods.
		return nil
	}

	cls, ok, err := c.getCluster(tenantID, clusterID)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %s", err)
	}
	if !ok {
		return fmt.Errorf("cluster not found: %s", clusterID)
	}
	cls.AssumedGPUPodsByKey[key] = &AssumedGPUPod{
		AllocatedCount: int32(gpuCount),
		AddedAt:        time.Now(),
	}
	c.logger.V(3).Info("added assumed pod", "key", key, "assumedPods", cls.AssumedGPUPodsByKey)

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
	return &Cluster{
		ClusterID:              c.ClusterID,
		ClusterName:            c.Name,
		UpdatedAt:              c.UpdatedAt,
		GPUNodes:               status.GpuNodes,
		ProvisionableResources: status.ProvisionableResources,
		GPUPods:                status.GpuPods,
		AssumedGPUPodsByKey:    make(map[string]*AssumedGPUPod),
	}, nil
}
