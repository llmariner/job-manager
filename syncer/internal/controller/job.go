package controller

import (
	"context"
	"fmt"
	"reflect"

	v1 "github.com/llmariner/job-manager/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	jobControllerName     = "job-controller"
	fullJobControllerName = domain + "/" + jobControllerName
)

var jobGVR = schema.GroupVersionResource{
	Group:    "batch",
	Version:  "v1",
	Resource: "jobs",
}

// JobController reconciles a Job object
type JobController struct {
	syncController
	recorder record.EventRecorder
}

// SetupWithManager sets up the controller with the Manager.
func (c *JobController) SetupWithManager(mgr ctrl.Manager, ssClient v1.SyncerServiceClient) error {
	c.recorder = mgr.GetEventRecorderFor(fullJobControllerName)
	c.k8sClient = mgr.GetClient()
	c.ssClient = ssClient
	c.controllerName = fullJobControllerName

	return ctrl.NewControllerManagedBy(mgr).
		Named(jobControllerName).
		For(&batchv1.Job{}).
		Complete(c)
}

// Reconcile reconciles a local Job object and deploys it to the worker cluster.
func (c *JobController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var job batchv1.Job
	if err := c.k8sClient.Get(ctx, req.NamespacedName, &job); err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			log.Error(err, "Failed to get job")
		}
		return ctrl.Result{}, err
	}

	if mgr := ptr.Deref(job.Spec.ManagedBy, ""); mgr != c.controllerName {
		log.V(4).Info("Skip job", "managedBy", mgr)
		return ctrl.Result{}, nil
	}

	if isDeleted(&job) {
		return c.syncDeleted(ctx, req, &job, log)
	}

	if result, err := c.tagWithFinalizer(ctx, &job, log); err != nil {
		return result, err
	}

	if isDeployed(&job) {
		log.V(1).Info("Job is already deployed")
		return ctrl.Result{}, nil
	}

	deployObj := job.DeepCopy()
	deployObj.ObjectMeta = metav1.ObjectMeta{
		Name:      job.Name,
		Namespace: job.Namespace,
		Labels:    job.Labels,
	}
	deployObj.Labels = attachDeployedByLabel(filterLabels(deployObj.Labels), jobControllerName)
	deployObj.Spec.ManagedBy = nil
	if deployObj.Spec.Selector != nil {
		filterLabels(deployObj.Spec.Selector.MatchLabels)
	}
	filterLabels(deployObj.Spec.Template.Labels)

	var totalGPUs uint32
	for _, container := range job.Spec.Template.Spec.Containers {
		if container.Resources.Limits != nil {
			if gpu, ok := container.Resources.Limits["nvidia.com/gpu"]; ok {
				totalGPUs += uint32(gpu.Value())
			}
		}
	}
	patchReq, err := prepareSyncPatchRequest(deployObj, totalGPUs, jobGVR, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	resp, patchErr := c.syncPatch(ctx, patchReq)
	if patchErr != nil {
		log.Error(patchErr, "Failed to patch job")
		patch := client.MergeFrom(&job)
		newJob := job.DeepCopy()

		// To share the error message with the user, update the job status here.
		// Until the job is created to the worker cluster, the job status is not updated.
		newCond := batchv1.JobCondition{
			Type:               "FailedClusterSchedule",
			Status:             corev1.ConditionTrue,
			LastProbeTime:      metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "FailedScheduling",
			Message:            patchErr.Error(),
		}
		updated := false
		for i, curCond := range newJob.Status.Conditions {
			if curCond.Type == newCond.Type {
				newCond.LastTransitionTime = curCond.LastTransitionTime
				newJob.Status.Conditions[i] = newCond
				updated = true
				break
			}
		}
		if !updated {
			newJob.Status.Conditions = append(newJob.Status.Conditions, newCond)
		}
		if !reflect.DeepEqual(job.Status, newJob.Status) {
			if err := c.k8sClient.Status().Patch(ctx, newJob, patch); err != nil {
				log.Error(err, "Failed to update status", "job", job.Name)
			}
		}
		return ctrl.Result{}, patchErr
	}
	log.V(2).Info("Patched job", "response", resp)

	patch := client.MergeFrom(&job)
	newJob := job.DeepCopy()
	if result, err := c.storeObjectData(ctx, newJob, resp, patch, log); err != nil {
		return result, err
	}

	c.recorder.Event(&job, "Normal", "Deployed", fmt.Sprintf("Job(%s) is deployed to the Cluster(%s)", resp.Uid, resp.ClusterId))
	log.Info("Deployed job")
	return ctrl.Result{}, nil
}
