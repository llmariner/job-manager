package controller

import (
	"context"
	"fmt"
	"reflect"
	"time"

	v1 "github.com/llmariner/job-manager/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	domain             = "cloudnatix.com"
	controllerName     = "job-controller"
	fullControllerName = domain + "/" + controllerName

	annoKeyClusterID  = domain + "/cluster-id"
	annoKeyUID        = domain + "/uid"
	annoKeyDeployedAt = domain + "/deployed-at"

	jobLabelKey = domain + "/deployed-by"
)

var excludeLabelKeys = map[string]struct{}{
	"batch.kubernetes.io/controller-uid": {},
	"batch.kubernetes.io/job-name":       {},
	"controller-uid":                     {},
	"job-name":                           {},
}

var jobGVR = schema.GroupVersionResource{
	Group:    "batch",
	Version:  "v1",
	Resource: "jobs",
}

// JobController reconciles a Job object
type JobController struct {
	recorder  record.EventRecorder
	k8sClient client.Client
	ssClient  v1.SyncerServiceClient
}

// SetupWithManager sets up the controller with the Manager.
func (c *JobController) SetupWithManager(mgr ctrl.Manager, ssClient v1.SyncerServiceClient) error {
	c.recorder = mgr.GetEventRecorderFor(fullControllerName)
	c.k8sClient = mgr.GetClient()
	c.ssClient = ssClient
	return ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
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

	if mgr := ptr.Deref(job.Spec.ManagedBy, ""); mgr != fullControllerName {
		log.V(4).Info("Skip job", "managedBy", mgr)
		return ctrl.Result{}, nil
	}

	if !job.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&job, fullControllerName) {
			return ctrl.Result{}, nil
		}

		clusterID := job.Annotations[annoKeyClusterID]
		if clusterID != "" {
			if _, err := c.ssClient.DeleteKubernetesObject(
				appendAuthorization(ctx),
				&v1.DeleteKubernetesObjectRequest{
					ClusterId: clusterID,
					Namespace: req.Namespace,
					Name:      req.Name,
					Group:     jobGVR.Group,
					Version:   jobGVR.Version,
					Resource:  jobGVR.Resource,
				}); err != nil {
				log.Error(err, "Failed to delete job")
				return ctrl.Result{}, err
			}
		} else {
			log.V(1).Info("Cluster ID not found, this job might not be deployed")
		}

		controllerutil.RemoveFinalizer(&job, fullControllerName)
		if err := c.k8sClient.Update(ctx, &job); err != nil {
			log.Error(err, "Failed to remove finalizer")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		log.Info("Job finalizer is removed")
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(&job, fullControllerName) {
		controllerutil.AddFinalizer(&job, fullControllerName)
		if err := c.k8sClient.Update(ctx, &job); err != nil {
			log.Error(err, "add finalizer")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	if v := job.Annotations[annoKeyDeployedAt]; v != "" {
		log.V(1).Info("Job is already deployed")
		return ctrl.Result{}, nil
	}

	deployObj := job.DeepCopy()
	deployObj.ObjectMeta = metav1.ObjectMeta{
		Name:      job.Name,
		Namespace: job.Namespace,
		Labels:    job.Labels,
	}
	for k := range deployObj.Labels {
		if _, ok := excludeLabelKeys[k]; ok {
			delete(deployObj.Labels, k)
		}
	}
	deployObj.Labels[jobLabelKey] = controllerName
	deployObj.Spec.ManagedBy = nil
	if deployObj.Spec.Selector != nil {
		for k := range deployObj.Spec.Selector.MatchLabels {
			if _, ok := excludeLabelKeys[k]; ok {
				delete(deployObj.Spec.Selector.MatchLabels, k)
			}
		}
	}
	for k := range deployObj.Spec.Template.Labels {
		if _, ok := excludeLabelKeys[k]; ok {
			delete(deployObj.Spec.Template.Labels, k)
		}
	}

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&deployObj)
	if err != nil {
		log.Error(err, "Failed to convert job to unstructured")
		return ctrl.Result{}, err
	}
	uobj := &unstructured.Unstructured{Object: obj}
	data, err := uobj.MarshalJSON()
	if err != nil {
		log.Error(err, "Failed to marshal job")
		return ctrl.Result{}, err
	}

	patchReq := &v1.PatchKubernetesObjectRequest{
		Namespace: job.Namespace,
		Name:      job.Name,
		Group:     jobGVR.Group,
		Version:   jobGVR.Version,
		Resource:  jobGVR.Resource,
		Data:      data,
	}
	var totalGPUs int
	for _, container := range job.Spec.Template.Spec.Containers {
		if container.Resources.Limits != nil {
			if gpu, ok := container.Resources.Limits["nvidia.com/gpu"]; ok {
				totalGPUs += int(gpu.Value())
			}
		}
	}
	if totalGPUs > 0 {
		patchReq.Resources = &v1.PatchKubernetesObjectRequest_Resources{
			GpuLimit: int32(totalGPUs),
		}
	}

	resp, patchErr := c.ssClient.PatchKubernetesObject(
		appendAuthorization(ctx),
		patchReq)
	if patchErr != nil {
		log.Error(patchErr, "Failed to patch job")
		newCond := batchv1.JobCondition{
			Type:               "FailedClusterSchedule",
			Status:             corev1.ConditionTrue,
			LastProbeTime:      metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "Failed to schedule the job to the worker cluster",
			Message:            patchErr.Error(),
		}
		patch := client.MergeFrom(&job)
		newJob := job.DeepCopy()
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
	if newJob.Annotations == nil {
		newJob.Annotations = make(map[string]string)
	}
	newJob.Annotations[annoKeyClusterID] = resp.ClusterId
	newJob.Annotations[annoKeyUID] = resp.Uid
	newJob.Annotations[annoKeyDeployedAt] = metav1.Now().UTC().Format(time.RFC3339)
	if err := c.k8sClient.Patch(ctx, newJob, patch); err != nil {
		log.Error(err, "Failed to update job")
		return ctrl.Result{}, err
	}

	c.recorder.Event(&job, "Normal", "Deployed", fmt.Sprintf("Job(%s) is deployed to the Cluster(%s)", resp.Uid, resp.ClusterId))
	log.Info("Deployed job")
	return ctrl.Result{}, nil
}
