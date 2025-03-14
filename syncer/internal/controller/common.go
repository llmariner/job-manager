package controller

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/llmariner/job-manager/api/v1"
	v2 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

const (
	domain = "cloudnatix.com"

	annoKeyUID        = domain + "/uid"
	annoKeyClusterID  = domain + "/cluster-id"
	annoKeyDeployedAt = domain + "/deployed-at"

	deployedByLabelKey = domain + "/deployed-by"
)

var excludeLabelKeys = map[string]struct{}{
	"batch.kubernetes.io/controller-uid": {},
	"batch.kubernetes.io/job-name":       {},
	"controller-uid":                     {},
	"job-name":                           {},
}

type syncController struct {
	controllerName string
	recorder       record.EventRecorder
	k8sClient      client.Client
	ssClient       v1.SyncerServiceClient
}

func (c syncController) syncDeleted(ctx context.Context, req controllerruntime.Request, obj client.Object, log logr.Logger) (controllerruntime.Result, error) {
	if !controllerutil.ContainsFinalizer(obj, c.controllerName) {
		return controllerruntime.Result{}, nil
	}

	clusterID := obj.GetAnnotations()[annoKeyClusterID]
	if clusterID != "" {
		if _, err := c.ssClient.DeleteKubernetesObject(
			appendAuthorization(ctx),
			&v1.DeleteKubernetesObjectRequest{
				ClusterId: clusterID,
				Namespace: req.Namespace,
				Name:      req.Name,
				Group:     jobSetGVR.Group,
				Version:   jobSetGVR.Version,
				Resource:  jobSetGVR.Resource,
			}); err != nil {
			log.Error(err, "Failed to delete", "object", obj.GetName(), "clusterID", clusterID)
			return controllerruntime.Result{}, err
		}
	} else {
		log.V(1).Info("Cluster ID not found, this object might not be deployed", "object", obj.GetName())
	}

	controllerutil.RemoveFinalizer(obj, c.controllerName)
	if err := c.k8sClient.Update(ctx, obj); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return controllerruntime.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("Finalizer is removed")
	return controllerruntime.Result{}, nil
}

func (c syncController) syncPatch(ctx context.Context, req *v1.PatchKubernetesObjectRequest) (*v1.PatchKubernetesObjectResponse, error) {
	return c.ssClient.PatchKubernetesObject(appendAuthorization(ctx), req)
}

func (c syncController) tagWithFinalizer(ctx context.Context, obj client.Object, log logr.Logger) (controllerruntime.Result, error) {
	if !controllerutil.ContainsFinalizer(obj, c.controllerName) {
		controllerutil.AddFinalizer(obj, c.controllerName)
		if err := c.k8sClient.Update(ctx, obj); err != nil {
			log.Error(err, "add finalizer")
			return controllerruntime.Result{}, client.IgnoreNotFound(err)
		}
	}
	return controllerruntime.Result{}, nil
}

func (c syncController) storeDeploymentData(ctx context.Context, obj client.Object, resp *v1.PatchKubernetesObjectResponse, patch client.Patch, log logr.Logger) (controllerruntime.Result, error) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[annoKeyClusterID] = resp.ClusterId
	annotations[annoKeyUID] = resp.Uid
	annotations[annoKeyDeployedAt] = v2.Now().UTC().Format(time.RFC3339)
	obj.SetAnnotations(annotations)
	if err := c.k8sClient.Patch(ctx, obj, patch); err != nil {
		log.Error(err, "Failed to set local deployment status", "object", obj.GetName())
		return controllerruntime.Result{}, err
	}
	return controllerruntime.Result{}, nil
}

func attachSchedulerErr(conditions []v2.Condition, patchErr error) []v2.Condition {
	// To share the error message with the user, update the jobSet status here.
	// Until the jobSet is created to the worker cluster, the jobSet status is not updated.
	newCond := v2.Condition{
		Type:               "FailedClusterSchedule",
		Status:             v2.ConditionTrue,
		LastTransitionTime: v2.Now(),
		Reason:             "FailedScheduling", // must match // ^[A-Za-z]([A-Za-z0-9_,:]*[A-Za-z0-9_])?$
		Message:            patchErr.Error(),
	}
	for i, curCond := range conditions {
		if curCond.Type == newCond.Type {
			newCond.LastTransitionTime = curCond.LastTransitionTime
			conditions[i] = newCond
			return conditions
		}
	}
	return append(conditions, newCond)
}

func attachDeployedByLabel(labels map[string]string, controllerName string) map[string]string {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[deployedByLabelKey] = controllerName
	return labels
}

func filterLabels(labels map[string]string) map[string]string {
	for k := range labels {
		if _, ok := excludeLabelKeys[k]; ok {
			delete(labels, k)
		}
	}
	return labels
}

func isDeployed(o client.Object) bool {
	return o.GetAnnotations()[annoKeyDeployedAt] != ""
}

func isDeleted(obj client.Object) bool {
	return !obj.GetDeletionTimestamp().IsZero()
}

func prepareSyncPatchRequest(
	deployObj client.Object,
	totalGPUs uint32,
	gvr schema.GroupVersionResource,
	log logr.Logger,
	mutators ...func(uobj *unstructured.Unstructured) error,
) (*v1.PatchKubernetesObjectRequest, error) {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&deployObj)
	if err != nil {
		log.Error(err, "Failed to convert job set to unstructured")
		return nil, err
	}
	uobj := &unstructured.Unstructured{Object: obj}
	for _, mutator := range mutators {
		if err := mutator(uobj); err != nil {
			return nil, err
		}
	}
	if err := removeCreationTime(uobj); err != nil {
		log.Error(err, "Failed to remove creationTime from metadata")
		return nil, err
	}

	data, err := uobj.MarshalJSON()
	if err != nil {
		log.Error(err, "Failed to marshal job set")
		return nil, err
	}

	patchReq := &v1.PatchKubernetesObjectRequest{
		Namespace: deployObj.GetNamespace(),
		Name:      deployObj.GetName(),
		Group:     gvr.Group,
		Version:   gvr.Version,
		Resource:  gvr.Resource,
		Data:      data,
	}
	if totalGPUs > 0 {
		patchReq.Resources = &v1.PatchKubernetesObjectRequest_Resources{
			GpuLimit: int32(totalGPUs),
		}
	}
	return patchReq, nil
}
