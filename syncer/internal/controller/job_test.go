package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr/testr"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcileJob(t *testing.T) {
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "test"},
	}
	myJobFixture := func(m ...func(job *batchv1.Job)) *batchv1.Job {
		return jobFixture(append(m, func(job *batchv1.Job) { job.Name, job.Namespace = req.Name, req.Namespace })...)
	}
	tests := []struct {
		name     string
		job      *batchv1.Job
		patchErr error

		wantErr    bool
		wantPatch  bool
		wantDelete bool
		assertFn   func(t *testing.T, job batchv1.Job)
	}{
		{
			name: "deploy",
			job:  myJobFixture(),
			assertFn: func(t *testing.T, job batchv1.Job) {
				assert.Contains(t, job.Annotations, annoKeyClusterID)
				assert.Contains(t, job.Annotations, annoKeyDeployedAt)
				assert.Contains(t, job.Annotations, annoKeyUID)
				assert.Contains(t, job.Finalizers, fullJobControllerName)
			},
			wantPatch: true,
		},
		{
			name: "no change",
			job: myJobFixture(func(job *batchv1.Job) {
				job.Finalizers = append(job.Finalizers, fullJobControllerName)
				job.Annotations = map[string]string{annoKeyDeployedAt: metav1.Now().Format(time.RFC3339)}
			}),
		},
		{
			name: "finalize",
			job: myJobFixture(func(job *batchv1.Job) {
				job.DeletionTimestamp = ptr.To(metav1.Now())
				job.Annotations = map[string]string{annoKeyClusterID: "cid"}
				job.Finalizers = append(job.Finalizers, fullJobControllerName)
			}),
			wantDelete: true,
		},
		{
			name: "already deleted",
		},
		{
			name:      "patch error",
			job:       myJobFixture(),
			patchErr:  errors.New("no schedulable cluster"),
			wantPatch: true,
			wantErr:   true,
			assertFn: func(t *testing.T, job batchv1.Job) {
				require.Len(t, job.Status.Conditions, 1)
				assert.Equal(t, "no schedulable cluster", job.Status.Conditions[0].Message)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var objs []runtime.Object
			if test.job != nil {
				objs = append(objs, test.job)
			}
			ssClient := &fakeSyncerServiceClient{patchErr: test.patchErr}
			jobCtr := JobController{
				syncController: syncController{
					controllerName: fullJobControllerName,
					k8sClient:      fake.NewFakeClient(objs...),
					ssClient:       ssClient,
				},
				recorder: record.NewFakeRecorder(5),
			}

			ctx := context.Background()
			ctx = ctrl.LoggerInto(ctx, testr.NewWithOptions(t, testr.Options{Verbosity: 8}))

			_, err := jobCtr.Reconcile(ctx, req)
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
				var gotJob batchv1.Job
				err = jobCtr.k8sClient.Get(ctx, req.NamespacedName, &gotJob)
				require.NoError(t, err)
				test.assertFn(t, gotJob)
			}
		})
	}
}

type fakeSyncerServiceClient struct {
	patchErr error

	patchCount int
	delCount   int
	listCount  int
}

func (s *fakeSyncerServiceClient) PatchKubernetesObject(ctx context.Context, in *v1.PatchKubernetesObjectRequest, opts ...grpc.CallOption) (*v1.PatchKubernetesObjectResponse, error) {
	s.patchCount++
	if s.patchErr != nil {
		return nil, s.patchErr
	}
	return &v1.PatchKubernetesObjectResponse{
		ClusterId: "cid",
		Uid:       "uid",
	}, nil
}

func (s *fakeSyncerServiceClient) DeleteKubernetesObject(ctx context.Context, in *v1.DeleteKubernetesObjectRequest, opts ...grpc.CallOption) (*v1.DeleteKubernetesObjectResponse, error) {
	s.delCount++
	return &v1.DeleteKubernetesObjectResponse{}, nil
}

func (s *fakeSyncerServiceClient) ListClusterIDs(ctx context.Context, in *v1.ListClusterIDsRequest, opts ...grpc.CallOption) (*v1.ListClusterIDsResponse, error) {
	s.listCount++
	return &v1.ListClusterIDsResponse{}, nil
}
