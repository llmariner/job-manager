package controller

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr/testr"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
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
		NamespacedName: types.NamespacedName{
			Namespace: "default",
			Name:      "test",
		},
	}

	createJob := func(mutateFn func(job *batchv1.Job)) *batchv1.Job {
		labels := map[string]string{
			"batch.kubernetes.io/controller-uid": "uid",
			"batch.kubernetes.io/job-name":       "job",
			"controller-uid":                     "uid",
			"job-name":                           "test",
			"custom":                             "test",
		}
		job := batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.Name,
				Namespace: req.Namespace,
				Labels:    labels,
			},
			Spec: batchv1.JobSpec{
				ManagedBy: ptr.To(fullControllerName),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"job-name": "test",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyNever,
						Containers: []corev1.Container{
							{
								Name:  "hello",
								Image: "busybox",
								Args:  []string{"echo", "test"},
							},
						},
					},
				},
			},
		}
		if mutateFn != nil {
			mutateFn(&job)
		}
		return &job
	}

	var tests = []struct {
		name string
		job  *batchv1.Job

		wantPatch  bool
		wantDelete bool
		assertFn   func(t *testing.T, job batchv1.Job)
	}{
		{
			name: "deploy",
			job:  createJob(nil),
			assertFn: func(t *testing.T, job batchv1.Job) {
				assert.Contains(t, job.Annotations, annoKeyClusterID)
				assert.Contains(t, job.Annotations, annoKeyDeployedAt)
				assert.Contains(t, job.Annotations, annoKeyUID)
				assert.Contains(t, job.Finalizers, fullControllerName)
			},
			wantPatch: true,
		},
		{
			name: "no change",
			job: createJob(func(job *batchv1.Job) {
				job.Finalizers = append(job.Finalizers, fullControllerName)
				job.Annotations = map[string]string{annoKeyDeployedAt: metav1.Now().Format(time.RFC3339)}
			}),
		},
		{
			name: "finalize",
			job: createJob(func(job *batchv1.Job) {
				job.DeletionTimestamp = ptr.To(metav1.Now())
				job.Annotations = map[string]string{annoKeyClusterID: "cid"}
				job.Finalizers = append(job.Finalizers, fullControllerName)
			}),
			wantDelete: true,
		},
		{
			name: "already deleted",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var objs []runtime.Object
			if test.job != nil {
				objs = append(objs, test.job)
			}
			ssClient := &fakeSyncerServiceClient{}
			jobCtr := JobController{
				recorder:  record.NewFakeRecorder(5),
				k8sClient: fake.NewFakeClient(objs...),
				ssClient:  ssClient,
			}

			ctx := context.Background()
			ctx = ctrl.LoggerInto(ctx, testr.NewWithOptions(t, testr.Options{Verbosity: 8}))

			_, err := jobCtr.Reconcile(ctx, req)
			assert.NoError(t, err)

			if test.wantPatch {
				assert.Equal(t, 1, ssClient.patchCount)
			}
			if test.wantDelete {
				assert.Equal(t, 1, ssClient.delCount)
			}

			if test.assertFn != nil {
				var gotJob batchv1.Job
				err = jobCtr.k8sClient.Get(ctx, req.NamespacedName, &gotJob)
				assert.NoError(t, err)
				test.assertFn(t, gotJob)
			}
		})
	}
}

type fakeSyncerServiceClient struct {
	patchCount int
	delCount   int
}

func (s *fakeSyncerServiceClient) PatchKubernetesObject(ctx context.Context, in *v1.PatchKubernetesObjectRequest, opts ...grpc.CallOption) (*v1.PatchKubernetesObjectResponse, error) {
	s.patchCount++
	return &v1.PatchKubernetesObjectResponse{
		ClusterId: "cid",
		Uid:       "uid",
	}, nil
}

func (s *fakeSyncerServiceClient) DeleteKubernetesObject(ctx context.Context, in *v1.DeleteKubernetesObjectRequest, opts ...grpc.CallOption) (*v1.DeleteKubernetesObjectResponse, error) {
	s.delCount++
	return &v1.DeleteKubernetesObjectResponse{}, nil
}
