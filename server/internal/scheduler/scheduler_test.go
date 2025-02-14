package scheduler

import (
	"testing"

	"github.com/go-logr/logr/testr"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/server/internal/cache"
	"github.com/llmariner/job-manager/server/internal/store"
	"github.com/llmariner/rbac-manager/pkg/auth"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestSchedule(t *testing.T) {
	const (
		tenantID = "tenant0"
	)

	tcs := []struct {
		name          string
		clusters      []*store.Cluster
		userInfo      *auth.UserInfo
		gpuCount      int
		prevClusterID string
		want          SchedulingResult
		wantErr       bool
	}{
		{
			name:     "no clusters",
			clusters: []*store.Cluster{},
			userInfo: &auth.UserInfo{
				TenantID: tenantID,
				AssignedKubernetesEnvs: []auth.AssignedKubernetesEnv{
					{
						ClusterID: "cluster0",
						Namespace: "namespace0",
					},
				},
			},
			gpuCount: 1,
			wantErr:  true,
		},
		{
			name: "assigned gpu cluster",
			clusters: []*store.Cluster{
				{
					ClusterID: "cluster0",
					TenantID:  tenantID,
					Status: marshalStatus(t, &v1.ClusterStatus{
						GpuNodes: []*v1.GpuNode{
							{
								ResourceName:     "nvidia.com/gpu",
								AllocatableCount: 1,
							},
						},
					}),
				},
			},
			userInfo: &auth.UserInfo{
				TenantID: tenantID,
				AssignedKubernetesEnvs: []auth.AssignedKubernetesEnv{
					{
						ClusterID: "cluster0",
						Namespace: "namespace0",
					},
				},
			},
			gpuCount: 1,
			want: SchedulingResult{
				ClusterID: "cluster0",
				Namespace: "namespace0",
			},
		},
		{
			name: "assigned without gpu",
			clusters: []*store.Cluster{
				{
					ClusterID: "cluster0",
					TenantID:  tenantID,
					Status:    marshalStatus(t, &v1.ClusterStatus{}),
				},
			},
			userInfo: &auth.UserInfo{
				TenantID: tenantID,
				AssignedKubernetesEnvs: []auth.AssignedKubernetesEnv{
					{
						ClusterID: "cluster0",
						Namespace: "namespace0",
					},
				},
			},
			gpuCount: 0,
			want: SchedulingResult{
				ClusterID: "cluster0",
				Namespace: "namespace0",
			},
		},
		{
			name: "unassigned gpu cluster",
			clusters: []*store.Cluster{
				{
					ClusterID: "cluster0",
					TenantID:  tenantID,
					Status: marshalStatus(t, &v1.ClusterStatus{
						GpuNodes: []*v1.GpuNode{
							{
								ResourceName:     "nvidia.com/gpu",
								AllocatableCount: 1,
							},
						},
					}),
				},
			},
			userInfo: &auth.UserInfo{
				TenantID: tenantID,
				AssignedKubernetesEnvs: []auth.AssignedKubernetesEnv{
					{
						ClusterID: "cluster1",
						Namespace: "namespace0",
					},
				},
			},
			gpuCount: 1,
			wantErr:  true,
		},
		{
			name: "unassigned to the same cluster",
			clusters: []*store.Cluster{
				{
					ClusterID: "cluster0",
					TenantID:  tenantID,
					Status:    marshalStatus(t, &v1.ClusterStatus{}),
				},
			},
			userInfo: &auth.UserInfo{
				TenantID: tenantID,
				AssignedKubernetesEnvs: []auth.AssignedKubernetesEnv{
					{
						ClusterID: "cluster0",
						Namespace: "namespace0",
					},
				},
			},
			gpuCount:      0,
			prevClusterID: "cluster0",
			wantErr:       true,
		},
		{
			name: "two gpu clusters",
			clusters: []*store.Cluster{
				{
					ClusterID: "cluster0",
					TenantID:  tenantID,
					Status: marshalStatus(t, &v1.ClusterStatus{
						GpuNodes: []*v1.GpuNode{
							{
								ResourceName:     "nvidia.com/gpu",
								AllocatableCount: 16,
							},
						},
					}),
				},
				{
					ClusterID: "cluster1",
					TenantID:  tenantID,
					Status: marshalStatus(t, &v1.ClusterStatus{
						GpuNodes: []*v1.GpuNode{
							{
								ResourceName:     "nvidia.com/gpu",
								AllocatableCount: 8,
							},
						},
					}),
				},
			},
			userInfo: &auth.UserInfo{
				TenantID: tenantID,
				AssignedKubernetesEnvs: []auth.AssignedKubernetesEnv{
					{
						ClusterID: "cluster0",
						Namespace: "namespace0",
					},
					{
						ClusterID: "cluster1",
						Namespace: "namespace1",
					},
				},
			},
			gpuCount: 1,
			want: SchedulingResult{
				ClusterID: "cluster0",
				Namespace: "namespace0",
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			st, tearDown := store.NewTest(t)
			defer tearDown()

			for _, c := range tc.clusters {
				_, err := st.CreateOrUpdateCluster(c)
				assert.NoError(t, err)
			}

			sched := New(cache.NewStore(st, testr.New(t)), testr.New(t))
			got, err := sched.Schedule(tc.userInfo, tc.prevClusterID, tc.gpuCount)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCanProvisionGPUs(t *testing.T) {
	tcs := []struct {
		name          string
		status        *cache.Cluster
		requestedGPUs int
		want          bool
	}{
		{
			name:          "no gpu nodes and no provisionable resources",
			status:        &cache.Cluster{},
			requestedGPUs: 1,
			want:          false,
		},
		{
			name: "gpu nodes",
			status: &cache.Cluster{
				GPUNodes: []*v1.GpuNode{
					{
						ResourceName:     "nvidia.com/gpu",
						AllocatableCount: 1,
					},
				},
				ProvisionableResources: []*v1.ProvisionableResource{},
			},
			requestedGPUs: 1,
			want:          true,
		},
		{
			name: "gpu nodes with unallocated gpus",
			status: &cache.Cluster{
				GPUNodes: []*v1.GpuNode{
					{
						ResourceName:     "nvidia.com/gpu",
						AllocatableCount: 8,
					},
				},
				GPUPods: []*v1.GpuPod{
					{AllocatedCount: 4},
				},
				ProvisionableResources: []*v1.ProvisionableResource{},
			},
			requestedGPUs: 2,
			want:          true,
		},
		{
			name: "gpu nodes with insufficient gpus",
			status: &cache.Cluster{
				GPUNodes: []*v1.GpuNode{
					{
						ResourceName:     "nvidia.com/gpu",
						AllocatableCount: 8,
					},
				},
				GPUPods: []*v1.GpuPod{
					{AllocatedCount: 7},
				},
				ProvisionableResources: []*v1.ProvisionableResource{},
			},
			requestedGPUs: 2,
			want:          false,
		},
		{
			name: "gpu nodes with insufficient gpus",
			status: &cache.Cluster{
				GPUNodes: []*v1.GpuNode{
					{
						ResourceName:     "nvidia.com/gpu",
						AllocatableCount: 8,
					},
				},
				GPUPods: []*v1.GpuPod{
					{AllocatedCount: 3},
				},
				AssumedGPUPodsByKey: map[string]*cache.AssumedGPUPod{
					"ns-0/pod-1": {AllocatedCount: 4},
				},
				ProvisionableResources: []*v1.ProvisionableResource{},
			},
			requestedGPUs: 2,
			want:          false,
		},
		{
			name: "provisionable resources with gpu instance type",
			status: &cache.Cluster{
				GPUNodes: []*v1.GpuNode{},
				ProvisionableResources: []*v1.ProvisionableResource{
					{
						InstanceType: "g5.12xlarge",
					},
				},
			},
			requestedGPUs: 1,
			want:          true,
		},
		{
			name: "provisionable resources with non-gpu instance type",
			status: &cache.Cluster{
				GPUNodes: []*v1.GpuNode{},
				ProvisionableResources: []*v1.ProvisionableResource{
					{
						InstanceType: "m5.12xlarge",
					},
				},
			},
			requestedGPUs: 1,
			want:          false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			s := S{logger: testr.New(t)}
			got, err := s.canProvisionGPUs(tc.requestedGPUs, tc.status)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsAWSInstanceTypeForNvidiaGPU(t *testing.T) {
	tcs := []struct {
		name     string
		instType string
		want     bool
		wantErr  bool
	}{
		{
			name:     "gpu instance type",
			instType: "g5.12xlarge",
			want:     true,
		},
		{
			name:     "non-gpu instance type",
			instType: "m5.12xlarge",
			want:     false,
		},
		{
			name:     "invalid instance type",
			instType: "invalid",
			wantErr:  true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, err := isAWSInstanceTypeForNvidiaGPU(tc.instType)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsAWSInstanceFamilyForNvidiaGPU(t *testing.T) {
	tcs := []struct {
		name       string
		instFamily string
		want       bool
		wantErr    bool
	}{
		{
			name:       "gpu",
			instFamily: "g5",
			want:       true,
		},
		{
			name:       "non-gpu",
			instFamily: "m5",
			want:       false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, err := isAWSInstanceFamilyForNvidiaGPU(tc.instFamily)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func marshalStatus(t *testing.T, status *v1.ClusterStatus) []byte {
	b, err := proto.Marshal(status)
	assert.NoError(t, err)
	return b
}
