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

	var (
		bestResult *SchedulingResult
		bestScore  float64
	)
	var infeasibleReasons []string
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

		score, err := s.scoreCluster(c, gpuCount)
		if err != nil {
			return SchedulingResult{}, err
		}

		s.logger.V(1).Info("Scheduled a workload", "clusterID", c.ClusterID, "namespace", ns)

		if !score.isFeasible {
			s.logger.V(1).Info("Ignoring a cluster as the workload cannot be scheduled", "clusterID", c.ClusterID)
			infeasibleReasons = append(infeasibleReasons, fmt.Sprintf("{cluster: %q, reason: %q}", c.ClusterName, score.infeasibleReason))
			continue
		}
		if bestResult == nil || score.score > bestScore {
			bestResult = &SchedulingResult{
				ClusterID:   c.ClusterID,
				ClusterName: c.ClusterName,
				Namespace:   ns,
			}
			bestScore = score.score
		}
	}

	if bestResult == nil {
		if len(infeasibleReasons) == 0 {
			return SchedulingResult{}, fmt.Errorf("no schedulable cluster")
		}

		return SchedulingResult{}, fmt.Errorf("workload not schedulable: %s", strings.Join(infeasibleReasons, ", "))
	}

	s.logger.V(1).Info("Scheduled a workload", "clusterID", bestResult.ClusterID, "namespace", bestResult.Namespace)
	return *bestResult, nil
}

type schedulingScore struct {
	isFeasible       bool
	score            float64
	infeasibleReason string
}

func (s *S) scoreCluster(c *cache.Cluster, requestedGPUs int) (schedulingScore, error) {
	if requestedGPUs == 0 {
		// Simply assume that the workload can run there.
		return schedulingScore{
			isFeasible: true,
			score:      0,
		}, nil
	}

	ok, err := s.canProvisionGPUs(requestedGPUs, c)
	if err != nil {
		return schedulingScore{}, err
	}
	if !ok {
		return schedulingScore{
			isFeasible:       false,
			infeasibleReason: "insufficient GPU resources",
		}, err
	}

	return schedulingScore{
		isFeasible: true,
		// TODO(kenji): Improve the scoring algorithm. Currently it is a simple best-fit where a capacility with the largest
		// number of available GPUs is selected.
		score: float64(availableGPUs(c)),
	}, nil
}

// canProvisionGPUs returns true if the cluster can provision GPUs.
//
// TODO(kenji): Support other cloud providers and non-Nvidia GPUs.
func (s *S) canProvisionGPUs(requestedGPUs int, c *cache.Cluster) (bool, error) {
	if len(c.GPUNodes) > 0 {
		// TODO(kenji): Take into resource fragmentation.
		avail := availableGPUs(c)
		s.logger.V(3).Info("Checking GPU resources", "requestedGPUs", requestedGPUs, "availableGPUs", avail)
		return requestedGPUs <= avail, nil
	}

	// TODO(guangrui): Consider to check if the instance type is GPU instance type.
	if len(c.ProvisionableResources) > 0 {
		return true, nil
	}
	return false, nil
}

func availableGPUs(c *cache.Cluster) int {
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

	return allocatable - allocated
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
