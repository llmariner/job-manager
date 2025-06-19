package clusterstatus

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/rbac-manager/pkg/auth"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	krpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
)

const (
	nvidiaGPU corev1.ResourceName = "nvidia.com/gpu"
)

type updater interface {
	UpdateClusterStatus(context.Context, *v1.UpdateClusterStatusRequest, ...grpc.CallOption) (*v1.UpdateClusterStatusResponse, error)
}

// NewManager creates a new manager.
func NewManager(
	updater updater,
	updateInterval time.Duration,
) *Manager {
	return &Manager{
		updater:        updater,
		updateInterval: updateInterval,
	}
}

// Manager is a manager.
type Manager struct {
	k8sClient      client.Client
	updater        updater
	updateInterval time.Duration
	logger         logr.Logger
}

// SetupWithManager sets up the updater with the manager.
func (m *Manager) SetupWithManager(mgr ctrl.Manager) error {
	m.k8sClient = mgr.GetClient()
	m.logger = mgr.GetLogger().WithName("clusterstatus")
	return mgr.Add(m)
}

// NeedLeaderElection implements LeaderElectionRunnable and always returns true.
func (m *Manager) NeedLeaderElection() bool {
	return true
}

// Start starts the manager.
func (m *Manager) Start(ctx context.Context) error {
	if err := m.updateClusterStaus(ctx); err != nil {
		return err
	}

	tick := time.NewTicker(m.updateInterval)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			if err := m.updateClusterStaus(ctx); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()

		}
	}
}

func (m *Manager) updateClusterStaus(ctx context.Context) error {
	m.logger.Info("Updating cluster status")

	status, err := m.buildClusterStaus(ctx)
	if err != nil {
		return err
	}

	ctx = auth.AppendWorkerAuthorization(ctx)
	req := &v1.UpdateClusterStatusRequest{
		ClusterStatus: status,
	}
	if _, err := m.updater.UpdateClusterStatus(ctx, req); err != nil {
		return err
	}

	m.logger.Info("Updated cluster status")

	return nil
}

func (m *Manager) buildClusterStaus(ctx context.Context) (*v1.ClusterStatus, error) {
	// TODO(kenji): Add pagination.

	nodes := &corev1.NodeList{}
	if err := m.k8sClient.List(ctx, nodes); err != nil {
		return nil, err
	}
	m.logger.Info("Found Nodes", "count", len(nodes.Items))
	var gpuNodes []*v1.GpuNode
	for _, node := range nodes.Items {
		if n, ok := toGPUNode(node, m.logger); ok {
			gpuNodes = append(gpuNodes, n)
		}
	}

	pods := &corev1.PodList{}
	if err := m.k8sClient.List(ctx, pods); err != nil {
		return nil, err
	}
	m.logger.Info("Found Pods", "count", len(pods.Items))
	var gpuPods []*v1.GpuPod
	for _, pod := range pods.Items {
		if p, ok := toGPUPod(&pod, m.logger); ok {
			gpuPods = append(gpuPods, p)
		}
	}

	var prs []*v1.ProvisionableResource
	nodePools := &krpv1.NodePoolList{}
	if err := m.k8sClient.List(ctx, nodePools); err != nil {
		// Ignore the error as this happens when the CRD is not installed.
	} else {
		m.logger.Info("Found NodePools", "count", len(nodePools.Items))
		for _, np := range nodePools.Items {
			prs = append(prs, toProvisionableResource(np))
		}
	}

	// TODO(kenji): Support Cluster Autoscaler and v1beta1 Karpenter.

	return &v1.ClusterStatus{
		GpuNodes:               gpuNodes,
		GpuPods:                gpuPods,
		ProvisionableResources: prs,
	}, nil
}

func toProvisionableResource(np krpv1.NodePool) *v1.ProvisionableResource {
	var instType, instFamily string

	for _, t := range np.Spec.Template.Spec.Requirements {
		if t.Operator != corev1.NodeSelectorOpIn {
			continue
		}
		if len(t.Values) == 0 {
			continue
		}

		v := t.Values[0]
		switch t.Key {
		case "karpenter.k8s.aws/instance-type":
			instType = v
		case "karpenter.k8s.aws/instance-family":
			instFamily = v
		}
	}

	return &v1.ProvisionableResource{
		InstanceType:   instType,
		InstanceFamily: instFamily,
	}
}

func toGPUNode(node corev1.Node, logger logr.Logger) (*v1.GpuNode, bool) {
	// Ignore cordoned nodes.
	if node.Spec.Unschedulable {
		return nil, false
	}

	// TODO(kenji): Support other accelerator types.
	rs := map[corev1.ResourceName]bool{
		nvidiaGPU: true,
	}

	for name, v := range node.Status.Allocatable {
		if !rs[name] {
			continue
		}

		count, ok := v.AsInt64()
		if !ok {
			logger.Info("Failed to convert to int64", "name", name.String(), "value", v.String())
			continue
		}

		return &v1.GpuNode{
			ResourceName: name.String(),
			// Cast to int32 is safe as one node cannot have such a large number of GPUs.
			AllocatableCount: int32(count),
		}, true
	}
	return nil, false
}

func toGPUPod(pod *corev1.Pod, logger logr.Logger) (*v1.GpuPod, bool) {
	// Ignore non-runing pods.
	if pod.Status.Phase != corev1.PodRunning {
		return nil, false
	}

	total := 0
	for _, con := range pod.Spec.Containers {
		limit := con.Resources.Limits
		if limit == nil {
			continue
		}

		// TODO(kenji): Support other accelerator types.
		v, ok := limit[nvidiaGPU]
		if !ok {
			continue
		}
		count, ok := v.AsInt64()
		if !ok {
			logger.Info("Failed to convert to int64", "value", v.String())
			continue
		}
		total += int(count)
	}

	if total == 0 {
		return nil, false
	}

	return &v1.GpuPod{
		ResourceName:   nvidiaGPU.String(),
		AllocatedCount: int32(total),
		NamespacedName: fmt.Sprintf("%s/%s", pod.Namespace, pod.Name),
	}, true
}
