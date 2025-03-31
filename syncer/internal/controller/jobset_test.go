package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	jobset "sigs.k8s.io/jobset/api/jobset/v1alpha2"
)

func TestReconcileJobSet(t *testing.T) {
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "myNamespace", Name: "myJobSet"},
	}
	myJobSetFixture := func(m ...func(js *jobset.JobSet)) *jobset.JobSet {
		return jobSetFixture(append(m, func(js *jobset.JobSet) {
			js.Name, js.Namespace = req.Name, req.Namespace
		})...,
		)
	}
	ctx := context.Background()
	ctx = ctrl.LoggerInto(ctx, testr.NewWithOptions(t, testr.Options{Verbosity: 8}))

	tests := map[string]struct {
		src      *jobset.JobSet
		patchErr error

		wantErr    bool
		expPatches int
		expDeletes int
		assertFn   func(t *testing.T, js jobset.JobSet)
	}{
		"deploy": {
			src: myJobSetFixture(),
			assertFn: func(t *testing.T, js jobset.JobSet) {
				assert.Contains(t, js.Annotations, annoKeyClusterID)
				assert.Contains(t, js.Annotations, annoKeyDeployedAt)
				assert.Contains(t, js.Annotations, annoKeyUID)
				assert.Contains(t, js.Finalizers, fullJobSetControllerName)
			},
			expPatches: 1,
		},
		"no change": {
			src: myJobSetFixture(func(js *jobset.JobSet) {
				js.Finalizers = append(js.Finalizers, fullJobSetControllerName)
				js.Annotations = map[string]string{annoKeyDeployedAt: metav1.Now().Format(time.RFC3339)}
			}),
		},
		"not managed by jobSet controller": {
			src: myJobSetFixture(func(js *jobset.JobSet) {
				js.Spec.ManagedBy = ptr.To("other")
			}),
		},
		"finalize": {
			src: myJobSetFixture(func(js *jobset.JobSet) {
				js.DeletionTimestamp = ptr.To(metav1.Now())
				js.Annotations = map[string]string{annoKeyClusterID: "cid"}
				js.Finalizers = append(js.Finalizers, fullJobSetControllerName)
			}),
			expDeletes: 1,
		},
		"already deleted": {},
		"patch error": {
			src:        myJobSetFixture(),
			patchErr:   errors.New("no schedulable cluster"),
			expPatches: 1,
			wantErr:    true,
			assertFn: func(t *testing.T, js jobset.JobSet) {
				require.Len(t, js.Status.Conditions, 1)
				assert.Equal(t, "no schedulable cluster", js.Status.Conditions[0].Message)
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithScheme(Scheme)
			if test.src != nil {
				fakeClient = fakeClient.WithObjects(test.src).WithStatusSubresource(test.src)
			}
			ssClient := &fakeSyncerServiceClient{patchErr: test.patchErr}
			jobSetCtr := JobSetController{
				syncController: syncController{
					controllerName: fullJobSetControllerName,
					k8sClient:      fakeClient.Build(),
					ssClient:       ssClient,
				},
				recorder: record.NewFakeRecorder(5),
			}

			// when
			_, err := jobSetCtr.Reconcile(ctx, req)
			if test.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, test.expPatches, ssClient.patchCount)
			require.Equal(t, test.expDeletes, ssClient.delCount)

			if test.assertFn != nil {
				var gotJobSet jobset.JobSet
				err = jobSetCtr.k8sClient.Get(ctx, req.NamespacedName, &gotJobSet)
				require.NoError(t, err)
				test.assertFn(t, gotJobSet)
			}
		})
	}
}

func TestCalcMinJobSetGPUs(t *testing.T) {
	specs := map[string]struct {
		jobSet *jobset.JobSet
		exp    uint32
	}{
		"single GPU": {
			jobSet: jobSetFixture(func(js *jobset.JobSet) {
				spec := &js.Spec.ReplicatedJobs[0].Template.Spec
				spec.Parallelism = ptr.To(int32(1))
				spec.Completions = ptr.To(int32(1))
				spec.Template.Spec.Containers[0].Resources.
					Limits["nvidia.com/gpu"] = *resource.NewQuantity(1, resource.DecimalSI)
			}),
			exp: 1,
		},
		"multiple GPU": {
			jobSet: jobSetFixture(func(js *jobset.JobSet) {
				spec := &js.Spec.ReplicatedJobs[0].Template.Spec
				spec.Parallelism = ptr.To(int32(1))
				spec.Completions = ptr.To(int32(1))
				spec.Template.Spec.Containers[0].Resources.
					Limits["nvidia.com/gpu"] = *resource.NewQuantity(2, resource.DecimalSI)
			}),
			exp: 2,
		},
		"multiple GPU parallel": {
			jobSet: jobSetFixture(func(js *jobset.JobSet) {
				spec := &js.Spec.ReplicatedJobs[0].Template.Spec
				spec.Parallelism = ptr.To(int32(2))
				spec.Completions = nil
				spec.Template.Spec.Containers[0].Resources.
					Limits["nvidia.com/gpu"] = *resource.NewQuantity(2, resource.DecimalSI)
			}),
			exp: 4,
		},
		"multiple jobs": {
			jobSet: jobSetFixture(func(js *jobset.JobSet) {
				js.Spec.ReplicatedJobs = append(js.Spec.ReplicatedJobs, jobset.ReplicatedJob{
					Template: batchv1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							Parallelism: ptr.To(int32(5)),
							Completions: nil,
							Template:    podTemplateSpecFixture(),
						},
					},
				})
			}),
			exp: 5,
		},
		"multiple containers": {
			jobSet: jobSetFixture(func(js *jobset.JobSet) {
				spec := &js.Spec.ReplicatedJobs[0].Template.Spec
				spec.Parallelism = ptr.To(int32(3))
				spec.Completions = nil
				spec.Template.Spec.Containers[0].Resources.
					Limits["nvidia.com/gpu"] = *resource.NewQuantity(2, resource.DecimalSI)
				spec.Template.Spec.Containers = append(spec.Template.Spec.Containers, corev1.Container{
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							"nvidia.com/gpu": *resource.NewQuantity(4, resource.DecimalSI),
						},
					},
				})
			}),
			exp: 12,
		},
		"no GPU": {
			jobSet: jobSetFixture(func(js *jobset.JobSet) {
				delete(js.Spec.ReplicatedJobs[0].Template.Spec.Template.Spec.Containers[0].Resources.Limits, "nvidia.com/gpu")
			}),
			exp: 0,
		},
		"parallel field not set ": {
			jobSet: jobSetFixture(func(js *jobset.JobSet) {
				spec := &js.Spec.ReplicatedJobs[0].Template.Spec
				spec.Parallelism = nil
				spec.Completions = nil
			}),
			exp: 1,
		},
		"no containers": {
			jobSet: jobSetFixture(func(js *jobset.JobSet) {
				clear(js.Spec.ReplicatedJobs[0].Template.Spec.Template.Spec.Containers)
			}),
			exp: 0,
		},
		"empty jobSet": {
			jobSet: &jobset.JobSet{},
			exp:    0,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			got := calcMinJobSetGPUs(spec.jobSet)
			assert.Equal(t, spec.exp, got)
		})
	}
}
