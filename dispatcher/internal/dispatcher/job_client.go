package dispatcher

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	// To embed the command template.
	_ "embed"

	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	"github.com/llm-operator/job-manager/dispatcher/internal/config"
	"github.com/llm-operator/job-manager/dispatcher/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	batchv1apply "k8s.io/client-go/applyconfigurations/batch/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	managedJobAnnotationKey = "llm-operator/managed-pod"
	jobIDAnnotationKey      = "llm-operator/job-id"

	kueueQueueNameLabelKey = "kueue.x-k8s.io/queue-name"

	jobTTL = time.Hour * 24
)

//go:embed cmd.tpl
var cmdTemplate string

// NewJobClient returns a new JobCreator.
func NewJobClient(
	k8sClient client.Client,
	jobConfig config.JobConfig,
	kueueConfig config.KueueConfig,
) *JobClient {
	return &JobClient{
		k8sClient:   k8sClient,
		jobConfig:   jobConfig,
		kueueConfig: kueueConfig,
	}
}

// JobClient operates a Kubernetes Job resource for a job.
type JobClient struct {
	k8sClient   client.Client
	jobConfig   config.JobConfig
	kueueConfig config.KueueConfig
}

func (p *JobClient) createJob(ctx context.Context, jobData *store.Job, presult *PreProcessResult) error {
	// TODO(kenji): Create a real fine-tuning job. See https://github.com/llm-operator/job-manager/tree/main/build/experiments/fine-tuning.
	log := ctrl.LoggerFrom(ctx)

	log.Info("Creating a k8s Job resource for a job")

	spec, err := p.jobSpec(jobData, presult)
	if err != nil {
		return err
	}

	namespace := p.getNamespace(jobData.TenantID)
	obj := batchv1apply.
		Job(util.GetK8sJobName(jobData.JobID), namespace).
		WithAnnotations(map[string]string{
			managedJobAnnotationKey: "true",
			jobIDAnnotationKey:      jobData.JobID}).
		WithSpec(spec)

	if p.kueueConfig.Enable {
		obj.WithLabels(map[string]string{
			kueueQueueNameLabelKey: p.getQueueName(namespace),
		})
	}

	uobj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return err
	}
	patch := &unstructured.Unstructured{Object: uobj}
	opts := &client.PatchOptions{FieldManager: "job-manager-dispatcher", Force: ptr.To(true)}
	return p.k8sClient.Patch(ctx, patch, client.Apply, opts)
}

func (p *JobClient) jobSpec(jobData *store.Job, presult *PreProcessResult) (*batchv1apply.JobSpecApplyConfiguration, error) {
	cmd, err := p.cmd(jobData, presult)
	if err != nil {
		return nil, err

	}

	container := corev1apply.Container().
		WithName("main").
		WithImage(fmt.Sprintf("%s:%s", p.jobConfig.Image, p.jobConfig.Version)).
		WithImagePullPolicy(p.jobConfig.ImagePullPolicy).
		WithCommand("/bin/bash", "-c", cmd).
		WithResources(p.res())
	podSpec := corev1apply.PodSpec().
		WithContainers(container).
		WithRestartPolicy(corev1.RestartPolicyNever)
	jobSpec := batchv1apply.JobSpec().
		WithTTLSecondsAfterFinished(int32(jobTTL.Seconds())).
		WithBackoffLimit(3).
		WithTemplate(corev1apply.PodTemplateSpec().
			WithSpec(podSpec))
	return jobSpec, nil
}

func (p *JobClient) res() *corev1apply.ResourceRequirementsApplyConfiguration {
	if p.jobConfig.NumGPUs == 0 {
		return nil
	}
	return corev1apply.ResourceRequirements().
		WithLimits(corev1.ResourceList{
			"nvidia.com/gpu": *resource.NewQuantity(int64(p.jobConfig.NumGPUs), resource.DecimalSI),
		})
}

func (p *JobClient) cmd(jobData *store.Job, presult *PreProcessResult) (string, error) {
	jobProto, err := jobData.V1Job()
	if err != nil {
		return "", err
	}

	t := template.Must(template.New("cmd").Parse(cmdTemplate))
	type Params struct {
		BaseModelName     string
		BaseModelURLs     map[string]string
		TrainingFileURL   string
		ValidationFileURL string
		OutputModelURL    string

		NumProcessors     int
		AdditionalSFTArgs string
	}
	numProcessors := 1
	if p.jobConfig.NumGPUs > 0 {
		numProcessors = p.jobConfig.NumGPUs
	}
	params := Params{
		BaseModelName:     jobProto.Model,
		BaseModelURLs:     presult.BaseModelURLs,
		TrainingFileURL:   presult.TrainingFileURL,
		ValidationFileURL: presult.ValidationFileURL,
		OutputModelURL:    presult.OutputModelURL,

		NumProcessors:     numProcessors,
		AdditionalSFTArgs: toAddtionalSFTArgs(jobProto),
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, &params); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (p *JobClient) getNamespace(orgID string) string {
	// TODO(aya): rethink the mapping method organization to namespace.
	// static mapping by configmap or set namespace to job data?
	return orgID
}

func (p *JobClient) getQueueName(namespace string) string {
	// TODO(aya): rethink how to get queue name
	return p.kueueConfig.DefaultQueueName
}

func toAddtionalSFTArgs(jobProto *v1.Job) string {
	hp := jobProto.Hyperparameters
	if hp == nil {
		return ""
	}
	args := []string{}
	if v := hp.BatchSize; v > 0 {
		args = append(args, fmt.Sprintf("--per_device_train_batch_size=%d", v))
	}
	if v := hp.LearningRateMultiplier; v > 0 {
		args = append(args, fmt.Sprintf("--learning_rate=%f", v))
	}
	if v := hp.NEpochs; v > 0 {
		args = append(args, fmt.Sprintf("--num_train_epochs=%d", v))
	}
	return strings.Join(args, " ")

}
func isManagedJob(annotations map[string]string) bool {
	return annotations[managedJobAnnotationKey] == "true"
}
