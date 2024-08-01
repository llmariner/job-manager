package dispatcher

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
	"time"

	"github.com/go-logr/logr"
	fv1 "github.com/llm-operator/file-manager/api/v1"
	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/dispatcher/internal/config"
	is3 "github.com/llm-operator/job-manager/dispatcher/internal/s3"
	"github.com/llm-operator/rbac-manager/pkg/auth"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	batchv1apply "k8s.io/client-go/applyconfigurations/batch/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	metav1apply "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	batchJobManagedAnnotationKey = "llm-operator/managed-batchjob"
	batchJobIDAnnotationKey      = "llm-operator/batchjob-id"

	bjManagerName = "batchjob-manager"
)

// TODO(aya): make configurable
const initImage = "mirror.gcr.io/alpine:3.10"

const (
	batchJobInitCmdTemplate = `set -xeuo pipefail
{{range $name, $url := .DataFileURLs}}
wget -q --output-document={{$.DataPath}}/{{$name}} "{{$url}}"
{{end}}
{{- if .MasterAddr }}
[ ${JOB_COMPLETION_INDEX} -eq 0 ] && exit 0
for i in $(seq 100); do
  nslookup {{.MasterAddr}}>/dev/null && exit 0
  sleep 2
done
{{- end }}`
	batchJobMainCmdTemplate = `set -xeuo pipefail
[ -f {{.ScriptsPath}}/requirements.txt ] && pip install -r {{.ScriptsPath}}/requirements.txt
{{.Command}}`
)

// BatchJobManagerOptions contains the options for the BatchJobManager.
type BatchJobManagerOptions struct {
	K8sClient  client.Client
	S3Client   s3Client
	FileClient fileClient
	BwClient   v1.BatchWorkerServiceClient

	LlmoBaseURL string
	ClusterID   string

	WandbConfig config.WandbAPIKeySecretConfig
	KueueConfig config.KueueConfig
}

// NewBatchJobManager returns a new batch job manager.
func NewBatchJobManager(opts BatchJobManagerOptions) *BatchJobManager {
	return &BatchJobManager{
		k8sClient:   opts.K8sClient,
		s3Client:    opts.S3Client,
		fileClient:  opts.FileClient,
		bwClient:    opts.BwClient,
		llmoBaseURL: opts.LlmoBaseURL,
		clusterID:   opts.ClusterID,
		wandbConfig: opts.WandbConfig,
		kueueConfig: opts.KueueConfig,
	}
}

// BatchJobManager is a manager of batch jobs.
type BatchJobManager struct {
	k8sClient  client.Client
	s3Client   s3Client
	fileClient fileClient
	bwClient   v1.BatchWorkerServiceClient

	llmoBaseURL string
	clusterID   string

	wandbConfig config.WandbAPIKeySecretConfig
	kueueConfig config.KueueConfig
}

// SetupWithManager registers the LifecycleManager with the manager.
func (m *BatchJobManager) SetupWithManager(mgr ctrl.Manager) error {
	filterByAnno := (predicate.NewPredicateFuncs(func(object client.Object) bool {
		return isManagedBatchJob(object.GetAnnotations())
	}))
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.Job{}, builder.WithPredicates(filterByAnno)).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger {
			if r != nil {
				return mgr.GetLogger().WithValues("batchjob", r.NamespacedName)
			}
			return mgr.GetLogger()
		}).
		Complete(m)
}

// Reconcile reconciles the batchJob deployment.
func (m *BatchJobManager) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var job batchv1.Job
	if err := m.k8sClient.Get(ctx, req.NamespacedName, &job); err != nil {
		log.V(2).Info("Failed to get the k8s job", "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !job.DeletionTimestamp.IsZero() {
		log.V(2).Info("k8s job is being deleted")
		return ctrl.Result{}, nil
	}

	jobID := req.Name
	ctx = auth.AppendWorkerAuthorization(ctx)
	ibjob, err := m.bwClient.GetInternalBatchJob(ctx, &v1.GetInternalBatchJobRequest{Id: jobID})
	if err != nil {
		log.Error(err, "Failed to get the batch job")
		return reconcile.Result{}, err
	}
	switch ibjob.State {
	case v1.InternalBatchJob_QUEUED:
		// internal job state is updated after k8s job creation,
		// so the reconciler may also receive an internal job in the queued state.
		if ibjob.QueuedAction != v1.InternalBatchJob_CREATING {
			// do nothing while dispatcher processes the job
			return ctrl.Result{}, nil
		}
	case v1.InternalBatchJob_RUNNING:
	case v1.InternalBatchJob_SUCCEEDED,
		v1.InternalBatchJob_FAILED,
		v1.InternalBatchJob_DELETED:
		// do nothing, already complete
		log.V(2).Info("Batch job is already completed", "state", ibjob.State)
		return ctrl.Result{}, nil
	case v1.InternalBatchJob_CANCELED:
		var (
			expired        bool
			expirationTime time.Time
		)
		for _, cond := range job.Status.Conditions {
			if cond.Type == batchv1.JobSuspended {
				expirationTime = cond.LastTransitionTime.Add(jobTTL)
				expired = time.Now().After(expirationTime)
				break
			}
		}
		if !expired {
			requeueAfter := time.Until(expirationTime)
			log.V(2).Info("Batch job is cancelled but not expired yet", "requeue-after", requeueAfter)
			return ctrl.Result{Requeue: true, RequeueAfter: requeueAfter}, nil
		}
		log.Info("Deleting the cancelled and expired k8s job")
		return ctrl.Result{}, m.k8sClient.Delete(ctx, &job)
	default:
		// unspecified or unknown are not valid states
		// this error could not be recovered by k8s reconciliation, so just log and return
		log.Error(fmt.Errorf("unexpected batch job state: %v", ibjob.State), "Job state is invalid")
		return ctrl.Result{}, nil
	}

	if job.Status.Succeeded == 0 && job.Status.Failed == 0 {
		// TODO(aya): check pod status, image pull error is not propagated to the job
		log.V(2).Info("K8s job is still running")
		return ctrl.Result{}, nil
	}

	upReq := &v1.UpdateBatchJobStateRequest{Id: jobID}
	if job.Status.Failed > 0 {
		for _, cond := range job.Status.Conditions {
			if cond.Type == batchv1.JobFailed {
				upReq.Reason = cond.Reason
				upReq.Message = cond.Message
				break
			}
		}
		upReq.State = v1.InternalBatchJob_FAILED
	} else {
		upReq.State = v1.InternalBatchJob_SUCCEEDED
	}
	if _, err := m.bwClient.UpdateBatchJobState(ctx, upReq); err != nil {
		log.Error(err, "Failed to update the batch job state")
		return ctrl.Result{}, err
	}

	log.Info("Finished processing job")
	return ctrl.Result{}, nil
}

// createBatchJob creates a k8s job for the internal batch job. The created k8s job has init and main containers.
// init container downloads data files from object storage, and stores them in an shared volume. main container
// precedes the user command with installing requirements packages.
func (m *BatchJobManager) createBatchJob(ctx context.Context, ibjob *v1.InternalBatchJob) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Creating k8s resources for a batch job")

	name := ibjob.Job.Id
	labels := map[string]string{
		"app.kubernetes.io/name":       "llmo-batch-job",
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/created-by": bjManagerName,
	}
	if m.kueueConfig.Enable {
		labels[kueueQueueNameLabelKey] = m.kueueConfig.DefaultQueueName
	}

	var initEnvs, envs []*corev1apply.EnvVarApplyConfiguration
	for k, v := range ibjob.Job.Envs {
		envs = append(envs, corev1apply.EnvVar().WithName(k).WithValue(v))
	}
	envs = append(envs, corev1apply.EnvVar().WithName("OPENAI_BASE_URL").WithValue(m.llmoBaseURL))
	if c := m.wandbConfig; c.Name != "" && c.Key != "" {
		envs = append(envs, corev1apply.EnvVar().
			WithName("WANDB_API_KEY").
			WithValueFrom(corev1apply.EnvVarSource().
				WithSecretKeyRef(corev1apply.SecretKeySelector().
					WithName(c.Name).
					WithKey(c.Key))))
	}

	replicas := int32(1)
	completionMode := batchv1.NonIndexedCompletion
	var subdomain, masterAddr string
	var ports []*corev1apply.ContainerPortApplyConfiguration

	if k := ibjob.Job.Kind; k != nil {
		switch t := k.Kind.(type) {
		case *v1.BatchJob_Kind_Pytorch:
			replicas = t.Pytorch.WorkerCount
			completionMode = batchv1.IndexedCompletion
			subdomain = name
			// Pod with index 0 (RANK=0) works as the master and the pod can be accessed by the headless service.
			// https://kubernetes.io/docs/tasks/job/job-with-pod-to-pod-communication/
			masterAddr = fmt.Sprintf("%[1]s-0.%[1]s", name)
			const portNum = 23456
			ports = append(ports,
				corev1apply.ContainerPort().
					WithName("master").
					WithContainerPort(portNum).
					WithProtocol(corev1.ProtocolTCP))
			envs = append(envs,
				corev1apply.EnvVar().WithName("MASTER_ADDR").WithValue(masterAddr),
				corev1apply.EnvVar().WithName("MASTER_PORT").WithValue(fmt.Sprintf("%d", portNum)),
				corev1apply.EnvVar().WithName("WORLD_SIZE").WithValue(fmt.Sprintf("%d", replicas)),
				corev1apply.EnvVar().WithName("RANK").
					WithValueFrom(corev1apply.EnvVarSource().
						WithFieldRef(corev1apply.ObjectFieldSelector().
							WithFieldPath("metadata.labels['batch.kubernetes.io/job-completion-index']"))))
		default:
			return fmt.Errorf("unsupported kind: %T", k)
		}
	}

	limit := corev1.ResourceList{}
	if r := ibjob.Job.Resources; r != nil && r.GpuCount > 0 {
		limit["nvidia.com/gpu"] = *resource.NewQuantity(int64(r.GpuCount), resource.DecimalSI)
	}
	resources := corev1apply.ResourceRequirements()
	if len(limit) > 0 {
		resources.WithLimits(limit)
	}

	dataFileURLs := make(map[string]string, len(ibjob.Job.DataFiles))
	for _, value := range ibjob.Job.DataFiles {
		name, url, err := m.getNameAndPresignedURL(ctx, value)
		if err != nil {
			return err
		}
		dataFileURLs[name] = url
	}

	const dataPath = "/data"
	const scriptsPath = "/scripts"
	var initScript bytes.Buffer
	if err := template.Must(template.New("init").
		Parse(batchJobInitCmdTemplate)).
		Execute(&initScript, struct {
			DataPath     string
			DataFileURLs map[string]string
			MasterAddr   string
		}{
			DataPath:     dataPath,
			DataFileURLs: dataFileURLs,
			MasterAddr:   masterAddr,
		}); err != nil {
		return err
	}
	var boostrapScript bytes.Buffer
	if err := template.Must(template.New("main").
		Parse(batchJobMainCmdTemplate)).
		Execute(&boostrapScript, struct {
			ScriptsPath string
			Command     string
		}{
			ScriptsPath: scriptsPath,
			Command:     ibjob.Job.Command,
		}); err != nil {
		return err
	}

	volumeMounts := []*corev1apply.VolumeMountApplyConfiguration{
		corev1apply.VolumeMount().WithName("share-volume").WithMountPath(dataPath),
		corev1apply.VolumeMount().WithName("scripts-volume").WithMountPath(scriptsPath),
	}

	jobConf := batchv1apply.
		Job(name, ibjob.Job.KubernetesNamespace).
		WithLabels(labels).
		WithAnnotations(map[string]string{
			batchJobManagedAnnotationKey: "true",
			batchJobIDAnnotationKey:      ibjob.Job.Id}).
		WithSpec(batchv1apply.JobSpec().
			WithTTLSecondsAfterFinished(int32(jobTTL.Seconds())).
			WithCompletionMode(completionMode).
			WithCompletions(replicas).
			WithParallelism(replicas).
			WithBackoffLimit(0).
			WithTemplate(corev1apply.PodTemplateSpec().
				WithSpec(corev1apply.PodSpec().
					WithSubdomain(subdomain).
					WithRestartPolicy(corev1.RestartPolicyNever).
					WithInitContainers(corev1apply.Container().
						WithName("init").
						WithImage(initImage).
						WithCommand("/bin/sh", "-c", initScript.String()).
						WithEnv(initEnvs...).
						WithVolumeMounts(volumeMounts...)).
					WithContainers(corev1apply.Container().
						WithName("main").
						WithImage(ibjob.Job.Image).
						WithCommand("/bin/bash", "-c", boostrapScript.String()).
						WithResources(resources).
						WithPorts(ports...).
						WithEnv(envs...).
						WithEnvFrom(corev1apply.EnvFromSource().
							WithSecretRef(corev1apply.SecretEnvSource().
								WithName(name))).
						WithVolumeMounts(volumeMounts...)).
					WithVolumes(
						corev1apply.Volume().
							WithName("share-volume").
							WithEmptyDir(corev1apply.EmptyDirVolumeSource()),
						corev1apply.Volume().
							WithName("scripts-volume").
							WithConfigMap(corev1apply.ConfigMapVolumeSource().
								WithName(name))))))

	kjob, err := m.applyObject(ctx, jobConf)
	if err != nil {
		return err
	}

	gvk := kjob.GetObjectKind().GroupVersionKind()
	ownerRef := metav1apply.OwnerReference().
		WithAPIVersion(gvk.GroupVersion().String()).
		WithKind(gvk.Kind).
		WithName(kjob.GetName()).
		WithUID(kjob.GetUID()).
		WithBlockOwnerDeletion(true).
		WithController(true)

	// Secret and ConfigMap are pre-created by server, and dispatcher only set the owner reference here.
	// TODO(aya): garbage collect orphaned secrets
	objs := []any{
		corev1apply.Secret(name, ibjob.Job.KubernetesNamespace).
			WithLabels(labels).
			WithOwnerReferences(ownerRef),
		corev1apply.ConfigMap(name, ibjob.Job.KubernetesNamespace).
			WithLabels(labels).
			WithOwnerReferences(ownerRef),
	}

	if masterAddr != "" {
		objs = append(objs,
			corev1apply.Service(name, ibjob.Job.KubernetesNamespace).
				WithLabels(labels).
				WithOwnerReferences(ownerRef).
				WithSpec(corev1apply.ServiceSpec().
					WithClusterIP(corev1.ClusterIPNone).
					WithSelector(map[string]string{"job-name": name})))
	}

	for _, obj := range objs {
		if _, err := m.applyObject(ctx, obj); err != nil {
			return err
		}
	}
	return nil
}

func (m *BatchJobManager) cancelBatchJob(ctx context.Context, ibjob *v1.InternalBatchJob) error {
	var kjob batchv1.Job
	if err := m.k8sClient.Get(ctx, types.NamespacedName{
		Name:      ibjob.Job.Id,
		Namespace: ibjob.Job.KubernetesNamespace,
	}, &kjob); err != nil {
		log := ctrl.LoggerFrom(ctx)
		log.V(2).Info("Failed to get the k8s job", "error", err)
		return client.IgnoreNotFound(err)
	}
	kjob.Spec.Suspend = ptr.To(true)
	return m.k8sClient.Update(ctx, &kjob, client.FieldOwner(bjManagerName))
}

func (m *BatchJobManager) deleteBatchJob(ctx context.Context, ibjob *v1.InternalBatchJob) error {
	var kjob batchv1.Job
	if err := m.k8sClient.Get(ctx, types.NamespacedName{
		Name:      ibjob.Job.Id,
		Namespace: ibjob.Job.KubernetesNamespace,
	}, &kjob); err != nil {
		log := ctrl.LoggerFrom(ctx)
		log.V(2).Info("Failed to get the k8s job", "error", err)
		return client.IgnoreNotFound(err)
	}
	return m.k8sClient.Delete(ctx, &kjob, client.PropagationPolicy(metav1.DeletePropagationBackground))
}

func (m *BatchJobManager) applyObject(ctx context.Context, applyConfig any) (client.Object, error) {
	opts := &client.PatchOptions{FieldManager: bjManagerName, Force: ptr.To(true)}
	uobj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(applyConfig)
	if err != nil {
		return nil, err
	}
	obj := &unstructured.Unstructured{Object: uobj}
	if err := m.k8sClient.Patch(ctx, obj, client.Apply, opts); err != nil {
		return nil, err
	}
	return obj, nil
}

func (m *BatchJobManager) getNameAndPresignedURL(ctx context.Context, fileID string) (string, string, error) {
	fresp, err := m.fileClient.GetFilePath(ctx, &fv1.GetFilePathRequest{
		Id: fileID,
	})
	if err != nil {
		return "", "", fmt.Errorf("get file path: %s", err)
	}
	url, err := m.s3Client.GeneratePresignedURL(fresp.Path, preSignedURLExpire, is3.RequestTypeGetObject)
	if err != nil {
		return "", "", fmt.Errorf("generate presigned url: %s", err)
	}
	return fresp.Filename, url, nil
}

func isManagedBatchJob(annotations map[string]string) bool {
	return annotations[batchJobManagedAnnotationKey] == "true"
}
