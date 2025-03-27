package controller

import (
	"context"
	"fmt"
	"reflect"

	v1 "github.com/llmariner/job-manager/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	jobset "sigs.k8s.io/jobset/api/jobset/v1alpha2"
)

const (
	jobSetControllerName     = "jobset-controller"
	fullJobSetControllerName = domain + "/" + jobSetControllerName
)

var jobSetGVR = schema.GroupVersionResource{
	Group:    "jobset.x-k8s.io",
	Version:  "v1alpha2",
	Resource: "jobsets",
}

// JobSetController reconciles a Job object
type JobSetController struct {
	syncController
	recorder record.EventRecorder
}

// SetupWithManager sets up the controller with the Manager.
func (c *JobSetController) SetupWithManager(mgr ctrl.Manager, ssClient v1.SyncerServiceClient) error {
	c.recorder = mgr.GetEventRecorderFor(fullJobSetControllerName)
	c.k8sClient = mgr.GetClient()
	c.ssClient = ssClient
	c.controllerName = fullJobSetControllerName
	return ctrl.NewControllerManagedBy(mgr).
		Named(jobSetControllerName).
		For(&jobset.JobSet{}).
		Complete(c)
}

// Reconcile reconciles a local Job object and deploys it to the worker cluster.
func (c *JobSetController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var jobSet jobset.JobSet
	if err := c.k8sClient.Get(ctx, req.NamespacedName, &jobSet); err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			log.Error(err, "Failed to get jobSet")
		}
		return ctrl.Result{}, err
	}

	if mgr := ptr.Deref(jobSet.Spec.ManagedBy, ""); mgr != c.controllerName {
		log.V(4).Info("Skip jobSet set", "managedBy", mgr)
		return ctrl.Result{}, nil
	}

	if isDeleted(&jobSet) {
		return c.syncDeleted(ctx, req, &jobSet, log)
	}

	if result, err := c.tagWithFinalizer(ctx, &jobSet, log); err != nil {
		return result, err
	}

	if isDeployed(&jobSet) {
		log.V(1).Info("Job set is already deployed")
		return ctrl.Result{}, nil
	}

	deployObj := jobSet.DeepCopy()
	deployObj.ObjectMeta = metav1.ObjectMeta{
		Name:      jobSet.Name,
		Namespace: jobSet.Namespace,
		Labels:    jobSet.Labels,
	}

	deployObj.Labels = attachDeployedByLabel(filterLabels(deployObj.Labels), jobSetControllerName)
	deployObj.Spec.ManagedBy = nil
	for _, job := range deployObj.Spec.ReplicatedJobs {
		filterLabels(job.Template.Spec.Template.Labels)
		if job.Template.Spec.Selector != nil {
			filterLabels(job.Template.Spec.Selector.MatchLabels)
		}
	}

	patchReq, err := prepareSyncPatchRequest(deployObj, calcMinJobSetGPUs(deployObj), jobSetGVR, log, removeCreationTime)
	if err != nil {
		return ctrl.Result{}, err
	}

	resp, patchErr := c.syncPatch(ctx, patchReq)
	if patchErr != nil {
		log.Error(patchErr, "Failed to patch jobSet")
		patch := client.MergeFrom(&jobSet)
		newJobSet := jobSet.DeepCopy()
		newJobSet.Status.Conditions = attachSchedulerErr(newJobSet.Status.Conditions, patchErr)
		if !reflect.DeepEqual(jobSet.Status, newJobSet.Status) {
			if err := c.k8sClient.Status().Patch(ctx, newJobSet, patch); err != nil {
				log.Error(err, "Failed to update status", "jobSet", jobSet.Name)
			}
		}
		return ctrl.Result{}, patchErr
	}
	log.V(2).Info("Patched job set", "response", resp)

	patch := client.MergeFrom(&jobSet)
	newJobSet := jobSet.DeepCopy()
	if result, err := c.storeObjectData(ctx, newJobSet, resp, patch, log); err != nil {
		return result, err
	}

	c.recorder.Event(&jobSet, "Normal", "Deployed", fmt.Sprintf("JobSet(%s) is deployed to the Cluster(%s)", resp.Uid, resp.ClusterId))
	log.Info("Deployed job set")
	return ctrl.Result{}, nil
}

// creationTimestamp fields are not declared in the schema for jobsets. patch updates fail on null values therefore
// they must be removed first:
// .spec.replicatedJobs.#.template.spec.template.metadata
// .spec.replicatedJobs.#.template.metadata
func removeCreationTime(uobj *unstructured.Unstructured) error {
	replicatedJobs, found, err := unstructured.NestedSlice(uobj.Object, "spec", "replicatedJobs")
	if err != nil {
		return fmt.Errorf("error accessing replicatedJobs: %w", err)
	}
	if !found {
		return nil
	}

	for _, job := range replicatedJobs {
		if jobMap, ok := job.(map[string]any); ok {
			unstructured.RemoveNestedField(jobMap, "template", "spec", "template", "metadata", "creationTimestamp")
			unstructured.RemoveNestedField(jobMap, "template", "metadata", "creationTimestamp")
		}
	}

	if err := unstructured.SetNestedSlice(uobj.Object, replicatedJobs, "spec", "replicatedJobs"); err != nil {
		return fmt.Errorf("error updating replicatedJobs in uobj: %w", err)
	}
	return nil
}

// find minimum GPUs required
func calcMinJobSetGPUs(deployObj *jobset.JobSet) uint32 {
	// quick solution that picks the most expensive job in terms of GPU as bottleneck
	var maxJobGPU uint32
	for _, job := range deployObj.Spec.ReplicatedJobs {
		jobSpec := job.Template.Spec
		for _, container := range jobSpec.Template.Spec.Containers {
			if container.Resources.Limits == nil {
				continue
			}
			if gpu, ok := container.Resources.Limits["nvidia.com/gpu"]; ok {
				parallelJobs := ptr.Deref(jobSpec.Parallelism, 1)
				mustCompleteCount := ptr.Deref(jobSpec.Completions, 1)
				instanceCount := min(mustCompleteCount, parallelJobs)
				gpuCount := uint32(gpu.Value()) * uint32(instanceCount)
				if gpuCount > maxJobGPU {
					maxJobGPU = gpuCount
				}
			}
		}
	}
	return maxJobGPU
}
