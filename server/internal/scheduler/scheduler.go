package scheduler

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/llmariner/job-manager/server/internal/cache"
	"github.com/llmariner/rbac-manager/pkg/auth"
)

const (
	// staleThreshold is the threshold for stale clusters. Clusters that have not been updated for longer than
	// this threshold are considered are excluded from scheduling.
	staleThreshold = 30 * time.Minute
)

// New creates a new scheduler.
func New(
	cache *cache.Store,
	logger logr.Logger,
) *S {
	return &S{
		cache:  cache,
		logger: logger,
	}
}

// S is a scheduler.
type S struct {
	cache  *cache.Store
	logger logr.Logger
}

// SchedulingResult is the result of scheduling a workload.
type SchedulingResult struct {
	ClusterID   string
	ClusterName string
	Namespace   string
}

// Schedule returns a Kubernetes cluster and a namespace where a workload can be scheduled.
// Currently it simply picks up one of the clusters that can provision GPU resources.
// The function returns an error if a workload is not schedulable.
// PrevScheduledClusterID is the cluster where the workload was previously scheduled. Schedule
// will not reschedule the workload to the same cluster.
// TODO(kenji): Improve the algorithm.
func (s *S) Schedule(userInfo *auth.UserInfo, prevScheduledClusterID string, gpuCount int) (SchedulingResult, error) {
	clusters, err := s.cache.ListClustersByTenantID(userInfo.TenantID)
	if err != nil {
		return SchedulingResult{}, err
	}
	if len(clusters) == 0 {
		return SchedulingResult{}, fmt.Errorf("no clusters")
	}

	if len(userInfo.AssignedKubernetesEnvs) == 0 {
		return SchedulingResult{}, fmt.Errorf("no assigned Kubernetes environments")
	}

	namespacesByCluster := map[string]string{}
	for _, env := range userInfo.AssignedKubernetesEnvs {
		namespacesByCluster[env.ClusterID] = env.Namespace
	}
	s.logger.V(1).Info("Scheduling a workload", "gpuCount", gpuCount, "assignedClustersEnvs", userInfo.AssignedKubernetesEnvs)
	for _, c := range clusters {
		if time.Since(c.UpdatedAt) > staleThreshold {
			s.logger.V(1).Info("Ignoring a stale cluster", "clusterID", c.ClusterID)
			continue
		}
		if c.ClusterID == prevScheduledClusterID {
			s.logger.V(1).Info("Skipping the previous cluster", "clusterID", c.ClusterID)
			continue
		}

		ns, ok := namespacesByCluster[c.ClusterID]
		if !ok {
			s.logger.V(1).Info("Ignoring a cluster that is not assigned to the user", "clusterID", c.ClusterID)
			continue
		}

		// Just pick up the first cluster that can provision GPU resources if gpuCount is > 0.
		// Otherwise just pick up the first cluster.
		if gpuCount > 0 {
			if ok, err := s.canProvisionGPUs(gpuCount, c); err != nil {
				return SchedulingResult{}, err
			} else if !ok {
				s.logger.V(1).Info("Ignoring a cluster that cannot provision GPUs", "clusterID", c.ClusterID)
				continue
			}
		}
		s.logger.V(1).Info("Scheduled a workload", "clusterID", c.ClusterID, "namespace", ns)
		return SchedulingResult{
			ClusterID:   c.ClusterID,
			ClusterName: c.ClusterName,
			Namespace:   ns,
		}, nil
	}

	return SchedulingResult{}, fmt.Errorf("no schedulable cluster")
}

// canProvisionGPUs returns true if the cluster can provision GPUs.
//
// TODO(kenji): Support other cloud providers and non-Nvidia GPUs.
func (s *S) canProvisionGPUs(requestedGPUs int, c *cache.Cluster) (bool, error) {
	if len(c.GPUNodes) > 0 {
		// TODO(kenji): Take into resource fragmentation.
		var allocatable int
		for _, n := range c.GPUNodes {
			allocatable += int(n.AllocatableCount)
		}
		var allocated int
		for _, p := range c.GPUPods {
			allocated += int(p.AllocatedCount)
		}
		for _, p := range c.AssumedGPUPodsByKey {
			allocated += int(p.AllocatedCount)
		}
		s.logger.V(3).Info("Checking GPU resources", "requestedGPUs", requestedGPUs, "allocatable", allocatable, "allocated", allocated)
		return requestedGPUs <= allocatable-allocated, nil
	}

	for _, pr := range c.ProvisionableResources {
		if i := pr.InstanceType; i != "" {
			if ok, err := isAWSInstanceTypeForNvidiaGPU(i); err != nil {
				return false, err
			} else if ok {
				return true, nil
			}
		}

		if i := pr.InstanceFamily; i != "" {
			if ok, err := isAWSInstanceFamilyForNvidiaGPU(i); err != nil {
				return false, err
			} else if ok {
				return true, nil
			}
		}
	}

	return false, nil
}

func isAWSInstanceTypeForNvidiaGPU(instType string) (bool, error) {
	// Get the family from the instance type.
	l := strings.Split(instType, ".")
	if len(l) != 2 {
		return false, fmt.Errorf("invalid instance type: %s", instType)
	}

	return isAWSInstanceFamilyForNvidiaGPU(l[0])
}

func isAWSInstanceFamilyForNvidiaGPU(instFamily string) (bool, error) {
	gpuInsts := map[string]bool{
		"g5":   true,
		"p4d":  true,
		"p4de": true,
		"p5":   true,
	}
	return gpuInsts[instFamily], nil
}
