package dispatcher

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/go-logr/stdr"
	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileBatchJob(t *testing.T) {
	var tests = []struct {
		name string

		state       v1.InternalBatchJob_State
		mutateJobFn func(job *batchv1.Job)

		wantUpdate      bool
		wantRequeue     bool
		wantState       v1.InternalBatchJob_State
		wantAssertJobFn func(t *testing.T, gotJob *batchv1.Job, err error)
	}{
		{
			name:  "still running",
			state: v1.InternalBatchJob_RUNNING,
		},
		{
			name:  "job failed",
			state: v1.InternalBatchJob_RUNNING,
			mutateJobFn: func(job *batchv1.Job) {
				job.Status.Failed = 1
			},
			wantState: v1.InternalBatchJob_FAILED,
		},
		{
			name:  "successfully completed",
			state: v1.InternalBatchJob_RUNNING,
			mutateJobFn: func(job *batchv1.Job) {
				job.Status.Succeeded = 1
			},
			wantState: v1.InternalBatchJob_SUCCEEDED,
		},
		{
			name:  "already succeeded job",
			state: v1.InternalBatchJob_SUCCEEDED,
		},
		{
			name:  "already failed job",
			state: v1.InternalBatchJob_FAILED,
		},
		{
			name:  "canceled job (not expired)",
			state: v1.InternalBatchJob_CANCELED,
			mutateJobFn: func(job *batchv1.Job) {
				job.Spec.Suspend = ptr.To(true)
				job.Status.Conditions = append(job.Status.Conditions, batchv1.JobCondition{
					Type:               batchv1.JobSuspended,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
				})
			},
			wantRequeue: true,
			wantAssertJobFn: func(t *testing.T, gotJob *batchv1.Job, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:  "canceled job (expired)",
			state: v1.InternalBatchJob_CANCELED,
			mutateJobFn: func(job *batchv1.Job) {
				expr := time.Now().Add(-jobTTL)
				job.Spec.Suspend = ptr.To(true)
				job.Status.Conditions = append(job.Status.Conditions, batchv1.JobCondition{
					Type:               batchv1.JobSuspended,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(expr),
				})
			},
			wantAssertJobFn: func(t *testing.T, gotJob *batchv1.Job, err error) {
				assert.True(t, apierrors.IsNotFound(err), "should be deleted")
			},
		},
		{
			name:  "unknown state",
			state: v1.InternalBatchJob_STATE_UNSPECIFIED,
			// just logged and return no error
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			job := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
			}
			if test.mutateJobFn != nil {
				test.mutateJobFn(job)
			}
			k8sClient := fake.NewFakeClient(job)
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      job.Name,
					Namespace: job.Namespace,
				},
			}
			bwClient := &fakeBatchWorkerServiceClient{
				jobs: []*v1.InternalBatchJob{
					{Job: &v1.BatchJob{Id: job.Name}, State: test.state},
				},
				updatedState: map[string]v1.InternalBatchJob_State{},
			}

			mgr := NewBatchJobManager(BatchJobManagerOptions{
				K8sClient: k8sClient,
				BwClient:  bwClient,
			})

			logger := log.New(&testLogWriter{t}, "TEST: ", 0)
			ctx := ctrl.LoggerInto(context.Background(), stdr.New(logger))
			stdr.SetVerbosity(4)

			result, err := mgr.Reconcile(ctx, req)
			assert.Equal(t, test.wantRequeue, result.Requeue, "requeue event")
			assert.NoError(t, err)

			gotState := bwClient.updatedState[job.Name]
			assert.Equal(t, test.wantState, gotState)

			var gotJob batchv1.Job
			err = k8sClient.Get(context.Background(), req.NamespacedName, &gotJob)
			if test.wantAssertJobFn != nil {
				test.wantAssertJobFn(t, &gotJob, err)
			}
		})
	}
}

func TestCancelBatchJob(t *testing.T) {
	const (
		name      = "test-job"
		namespace = "default"
	)
	mgr := NewBatchJobManager(BatchJobManagerOptions{
		K8sClient: fake.NewFakeClient(&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		})})

	logger := log.New(&testLogWriter{t}, "TEST: ", 0)
	ctx := ctrl.LoggerInto(context.Background(), stdr.New(logger))
	stdr.SetVerbosity(4)

	err := mgr.cancelBatchJob(ctx, &v1.InternalBatchJob{
		Job: &v1.BatchJob{
			Id:                  name,
			KubernetesNamespace: namespace,
		},
	})
	assert.NoError(t, err)

	var updatedJob batchv1.Job
	err = mgr.k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &updatedJob)
	assert.NoError(t, err)
	assert.True(t, updatedJob.Spec.Suspend != nil && *updatedJob.Spec.Suspend, "suspended: %+v", updatedJob.Spec.Suspend)
}

type testLogWriter struct {
	t *testing.T
}

func (w *testLogWriter) Write(p []byte) (n int, err error) {
	w.t.Log(string(p))
	return len(p), nil
}
