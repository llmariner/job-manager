package dispatcher

import (
	"context"
	"testing"
	"time"

	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileJob(t *testing.T) {
	var tests = []struct {
		name string

		state       v1.InternalJob_State
		mutateJobFn func(job *batchv1.Job)

		wantErr         bool
		wantRequeue     bool
		wantUpdatePhase v1.UpdateJobPhaseRequest_Phase
		wantAssertJobFn func(t *testing.T, gotJob *batchv1.Job, err error)
	}{
		{
			name:  "still running",
			state: v1.InternalJob_RUNNING,
		},
		{
			name:  "job failed",
			state: v1.InternalJob_RUNNING,
			mutateJobFn: func(job *batchv1.Job) {
				job.Status.Failed = 1
			},
			wantUpdatePhase: v1.UpdateJobPhaseRequest_FAILED,
		},
		{
			name:  "successfully completed",
			state: v1.InternalJob_RUNNING,
			mutateJobFn: func(job *batchv1.Job) {
				job.Status.Succeeded = 1
			},
			wantUpdatePhase: v1.UpdateJobPhaseRequest_FINETUNED,
		},
		{
			name:  "already succeeded job",
			state: v1.InternalJob_SUCCEEDED,
		},
		{
			name:  "already failed job",
			state: v1.InternalJob_FAILED,
		},
		{
			name:        "canceled job (not expired)",
			state:       v1.InternalJob_CANCELED,
			wantRequeue: true,
			wantAssertJobFn: func(t *testing.T, gotJob *batchv1.Job, err error) {
				assert.NoError(t, err)
				assert.True(t, *gotJob.Spec.Suspend)
			},
		},
		{
			name:  "canceled job (expired)",
			state: v1.InternalJob_CANCELED,
			mutateJobFn: func(job *batchv1.Job) {
				expr := time.Now().Add(-jobTTL)
				job.Status.Conditions = append(job.Status.Conditions, batchv1.JobCondition{
					Type:               batchv1.JobSuspended,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(expr),
				})
			},
			wantAssertJobFn: func(t *testing.T, gotJob *batchv1.Job, err error) {
				assert.True(t, apierrors.IsNotFound(err))
			},
		},
		{
			name:  "unknown state",
			state: v1.InternalJob_UNSPECIFIED,
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

			ijobs := []*v1.InternalJob{
				{
					Job:   &v1.Job{Id: job.Name},
					State: test.state,
				},
			}
			wsClient := &fakeFineTuningWorkerServiceClient{
				jobs:          ijobs,
				updatedPhases: map[string]v1.UpdateJobPhaseRequest_Phase{},
			}

			mgr := NewLifecycleManager(wsClient, k8sClient, &NoopPostProcessor{})
			result, err := mgr.Reconcile(context.Background(), req)
			assert.Equal(t, test.wantRequeue, result.Requeue, "requeue event")
			if test.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			gotPhase := wsClient.updatedPhases[job.Name]
			assert.Equal(t, test.wantUpdatePhase, gotPhase)

			var gotJob batchv1.Job
			err = k8sClient.Get(context.Background(), req.NamespacedName, &gotJob)
			if test.wantAssertJobFn != nil {
				test.wantAssertJobFn(t, &gotJob, err)
			}
		})
	}
}
