package scheduler

import (
	"testing"

	"github.com/go-logr/logr/testr"
	v1 "github.com/llmariner/job-manager/api/v1"
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
		name     string
		clusters []*store.Cluster
		userInfo *auth.UserInfo
		want     SchedulingResult
		wantErr  bool
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
			wantErr: true,
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
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			st, tearDown := store.NewTest(t)
			defer tearDown()

			for _, c := range tc.clusters {
				err := st.CreateOrUpdateCluster(c)
				assert.NoError(t, err)
			}

			sched := New(st, testr.New(t))
			got, err := sched.Schedule(tc.userInfo)
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
		name   string
		status *v1.ClusterStatus
		want   bool
	}{
		{
			name: "no gpu nodes and no provisionable resources",
			status: &v1.ClusterStatus{
				GpuNodes:               []*v1.GpuNode{},
				ProvisionableResources: []*v1.ProvisionableResource{},
			},
			want: false,
		},
		{
			name: "gpu nodes",
			status: &v1.ClusterStatus{
				GpuNodes: []*v1.GpuNode{
					{
						ResourceName:     "nvidia.com/gpu",
						AllocatableCount: 1,
					},
				},
				ProvisionableResources: []*v1.ProvisionableResource{},
			},
			want: true,
		},
		{
			name: "provisionable resources with gpu instance type",
			status: &v1.ClusterStatus{
				GpuNodes: []*v1.GpuNode{},
				ProvisionableResources: []*v1.ProvisionableResource{
					{
						InstanceType: "g5.12xlarge",
					},
				},
			},
			want: true,
		},
		{
			name: "provisionable resources with non-gpu instance type",
			status: &v1.ClusterStatus{
				GpuNodes: []*v1.GpuNode{},
				ProvisionableResources: []*v1.ProvisionableResource{
					{
						InstanceType: "m5.12xlarge",
					},
				},
			},
			want: false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, err := canProvisionGPUs(tc.status)
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
