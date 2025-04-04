package clusterstatus

import (
	"context"
	"testing"

	"github.com/go-logr/logr/testr"
	"github.com/google/go-cmp/cmp"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	krpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
)

func TestManager(t *testing.T) {
	tcs := []struct {
		name string
		objs []runtime.Object
		want *v1.ClusterStatus
	}{
		{
			name: "gpu nodes",
			objs: []runtime.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1",
					},
					Status: corev1.NodeStatus{
						Allocatable: corev1.ResourceList{
							nvidiaGPU: resource.MustParse("1"),
						},
					},
				},
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node2",
					},
					Status: corev1.NodeStatus{
						Allocatable: corev1.ResourceList{
							nvidiaGPU: resource.MustParse("2"),
						},
					},
				},
			},
			want: &v1.ClusterStatus{
				GpuNodes: []*v1.GpuNode{
					{
						ResourceName:     nvidiaGPU.String(),
						AllocatableCount: 1,
					},
					{
						ResourceName:     nvidiaGPU.String(),
						AllocatableCount: 2,
					},
				},
			},
		},
		{
			name: "no gpu nodes",
			objs: []runtime.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1",
					},
					Status: corev1.NodeStatus{
						Allocatable: corev1.ResourceList{
							"cpu": resource.MustParse("1"),
						},
					},
				},
			},
			want: &v1.ClusterStatus{
				GpuNodes: []*v1.GpuNode{},
			},
		},
		{
			name: "cordoned gpu node",
			objs: []runtime.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1",
					},
					Spec: corev1.NodeSpec{
						Unschedulable: true,
					},
					Status: corev1.NodeStatus{
						Allocatable: corev1.ResourceList{
							nvidiaGPU: resource.MustParse("1"),
						},
					},
				},
			},
			want: &v1.ClusterStatus{
				GpuNodes: []*v1.GpuNode{},
			},
		},
		{
			name: "provisionable resources of instance type",
			objs: []runtime.Object{
				&krpv1.NodePool{
					Spec: krpv1.NodePoolSpec{
						Template: krpv1.NodeClaimTemplate{
							Spec: krpv1.NodeClaimTemplateSpec{
								Requirements: []krpv1.NodeSelectorRequirementWithMinValues{
									{
										NodeSelectorRequirement: corev1.NodeSelectorRequirement{
											Key:      "karpenter.k8s.aws/instance-type",
											Values:   []string{"g5.4xlarge"},
											Operator: corev1.NodeSelectorOpIn,
										},
									},
								},
							},
						},
					},
				},
			},
			want: &v1.ClusterStatus{
				ProvisionableResources: []*v1.ProvisionableResource{
					{
						InstanceType: "g5.4xlarge",
					},
				},
			},
		},
		{
			name: "provisionable resources of instance family",
			objs: []runtime.Object{
				&krpv1.NodePool{
					Spec: krpv1.NodePoolSpec{
						Template: krpv1.NodeClaimTemplate{
							Spec: krpv1.NodeClaimTemplateSpec{
								Requirements: []krpv1.NodeSelectorRequirementWithMinValues{
									{
										NodeSelectorRequirement: corev1.NodeSelectorRequirement{
											Key:      "karpenter.k8s.aws/instance-family",
											Values:   []string{"g5"},
											Operator: corev1.NodeSelectorOpIn,
										},
									},
								},
							},
						},
					},
				},
			},
			want: &v1.ClusterStatus{
				ProvisionableResources: []*v1.ProvisionableResource{
					{
						InstanceFamily: "g5",
					},
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			m := &Manager{
				k8sClient: fake.NewFakeClient(tc.objs...),
				logger:    testr.New(t),
			}
			got, err := m.buildClusterStaus(context.Background())
			assert.NoError(t, err)
			assert.Truef(t, proto.Equal(tc.want, got), cmp.Diff(tc.want, got, protocmp.Transform()))
		})
	}
}

func TestToGPUPod(t *testing.T) {
	tcs := []struct {
		name   string
		pod    *corev1.Pod
		want   *v1.GpuPod
		wantOK bool
	}{
		{
			name: "gpu pod",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									nvidiaGPU: resource.MustParse("1"),
								},
							},
						},
						{
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									nvidiaGPU: resource.MustParse("2"),
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
			want: &v1.GpuPod{
				ResourceName:   nvidiaGPU.String(),
				AllocatedCount: 3,
				NamespacedName: "default/pod1",
			},
			wantOK: true,
		},
		{
			name: "non-gpu pod",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"cpu": resource.MustParse("1"),
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
			wantOK: false,
		},
		{
			name: "pending gpu pod",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									nvidiaGPU: resource.MustParse("1"),
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
				},
			},
			wantOK: false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := toGPUPod(tc.pod, testr.New(t))
			if !tc.wantOK {
				assert.False(t, ok)
				return
			}
			assert.True(t, ok)
			assert.Truef(t, proto.Equal(tc.want, got), cmp.Diff(tc.want, got, protocmp.Transform()))
		})
	}
}
