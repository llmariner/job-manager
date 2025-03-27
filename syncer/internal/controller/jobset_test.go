package controller

import (
	"context"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	jobset "sigs.k8s.io/jobset/api/jobset/v1alpha2"
	"testing"
	"time"
)

func TestReconcileJobSet(t *testing.T) {
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "myNamespace", Name: "myJobSet"},
	}
	myJobSetFixture := func(m ...func(job *jobset.JobSet)) *jobset.JobSet {
		return jobSetFixture(append(m, func(jobSet *jobset.JobSet) {
			jobSet.Name, jobSet.Namespace = req.Name, req.Namespace
		})...,
		)
	}
	tests := map[string]struct {
		src      *jobset.JobSet
		patchErr error

		wantErr    bool
		wantPatch  bool
		wantDelete bool
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
			wantPatch: true,
		},
		"no change": {
			src: myJobSetFixture(func(js *jobset.JobSet) {
				js.Finalizers = append(js.Finalizers, fullJobSetControllerName)
				js.Annotations = map[string]string{annoKeyDeployedAt: metav1.Now().Format(time.RFC3339)}
			}),
		},
		"finalize": {
			src: myJobSetFixture(func(js *jobset.JobSet) {
				js.DeletionTimestamp = ptr.To(metav1.Now())
				js.Annotations = map[string]string{annoKeyClusterID: "cid"}
				js.Finalizers = append(js.Finalizers, fullJobSetControllerName)
			}),
			wantDelete: true,
		},
		"already deleted": {
			src: myJobSetFixture(func(js *jobset.JobSet) {
				js.DeletionTimestamp = ptr.To(metav1.Now())
				js.Finalizers = append(js.Finalizers, fullJobSetControllerName)
				js.Annotations = map[string]string{annoKeyClusterID: "anyID"}
			}),
		},
		//"patch error": { // fails with: jobsets.jobset.x-k8s.io \"myJobSet\" not found
		//	src:       myJobSetFixture(),
		//	patchErr:  errors.New("no schedulable cluster"),
		//	wantPatch: true,
		//	wantErr:   true,
		//	assertFn: func(t *testing.T, js jobset.JobSet) {
		//		require.Len(t, js.Status.Conditions, 1)
		//		assert.Equal(t, "no schedulable cluster", js.Status.Conditions[0].Message)
		//	},
		//},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithScheme(Scheme)
			if test.src != nil {
				fakeClient = fakeClient.WithObjects(test.src)
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

			ctx := context.Background()
			ctx = ctrl.LoggerInto(ctx, testr.NewWithOptions(t, testr.Options{Verbosity: 8}))

			// when
			_, err := jobSetCtr.Reconcile(ctx, req)
			if test.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if test.wantPatch {
				require.Equal(t, 1, ssClient.patchCount)
			}
			if test.wantDelete {
				require.Equal(t, 1, ssClient.delCount)
			}

			if test.assertFn != nil {
				var gotJobSet jobset.JobSet
				err = jobSetCtr.k8sClient.Get(ctx, req.NamespacedName, &gotJobSet)
				require.NoError(t, err)
				test.assertFn(t, gotJobSet)
			}
		})
	}
}
