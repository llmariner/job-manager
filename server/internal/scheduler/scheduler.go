package scheduler

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/llmariner/job-manager/server/internal/store"
	"github.com/llmariner/rbac-manager/pkg/auth"
)

// New creates a new scheduler.
func New(
	store *store.S,
	logger logr.Logger,
) *S {
	return &S{
		store:  store,
		logger: logger,
	}
}

// S is a scheduler.
type S struct {
	store  *store.S
	logger logr.Logger
}

// SchedulingResult is the result of scheduling a workload.
type SchedulingResult struct {
	ClusterID string
	Namespace string
}

// Schedule returns a Kubernetes cluster and a namespace where a workload can be scheduled.
// Currently it simply picks up one of the clusters that can provision GPU resources.
// The function returns an error if a workload is not schedulable.
//
// TODO(kenji): Implement.
func (s *S) Schedule(userInfo *auth.UserInfo) (SchedulingResult, error) {
	if len(userInfo.AssignedKubernetesEnvs) == 0 {
		return SchedulingResult{}, fmt.Errorf("no kuberentes cluster/namespace")
	}
	kenv := userInfo.AssignedKubernetesEnvs[0]
	return SchedulingResult{
		ClusterID: kenv.ClusterID,
		Namespace: kenv.Namespace,
	}, nil
}
