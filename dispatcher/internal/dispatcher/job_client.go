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
	"github.com/llm-operator/job-manager/dispatcher/internal/config"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

	jobManagerName = "job-manager-dispatcher"

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

func (p *JobClient) createJob(ctx context.Context, ijob *v1.InternalJob, presult *PreProcessResult) error {
	// TODO(kenji): Create a real fine-tuning job. See https://github.com/llm-operator/job-manager/tree/main/build/experiments/fine-tuning.
	log := ctrl.LoggerFrom(ctx)
	log.Info("Creating a k8s Job resource for a job")

	spec, err := p.jobSpec(ijob.Job, presult)
	if err != nil {
		return err
	}

	obj := batchv1apply.
		Job(ijob.Job.Id, ijob.Job.KubernetesNamespace).
		WithAnnotations(map[string]string{
			managedJobAnnotationKey: "true",
			jobIDAnnotationKey:      ijob.Job.Id}).
		WithSpec(spec)

	if p.kueueConfig.Enable {
		obj.WithLabels(map[string]string{
			kueueQueueNameLabelKey: p.getQueueName(ijob.Job.KubernetesNamespace),
		})
	}

	uobj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return err
	}
	patch := &unstructured.Unstructured{Object: uobj}
	opts := &client.PatchOptions{FieldManager: jobManagerName, Force: ptr.To(true)}
	return p.k8sClient.Patch(ctx, patch, client.Apply, opts)
}

func (p *JobClient) jobSpec(job *v1.Job, presult *PreProcessResult) (*batchv1apply.JobSpecApplyConfiguration, error) {
	cmd, err := p.cmd(job, presult)
	if err != nil {
		return nil, err
	}

	container := corev1apply.Container().
		WithName("main").
		WithImage(fmt.Sprintf("%s:%s", p.jobConfig.Image, p.jobConfig.Version)).
		WithImagePullPolicy(p.jobConfig.ImagePullPolicy).
		WithCommand("/bin/bash", "-c", cmd).
		WithResources(p.res())

	if s := p.jobConfig.WandbAPIKeySecret; s.Name != "" {
		if s.Key == "" {
			return nil, fmt.Errorf("wandb secret key is not set")
		}

		// TODO(kenji): Injecting the WANDB_API_KEY environment variable is
		// required to access, but ideally we should avoid exposing the secret to the job.
		container = container.WithEnv(corev1apply.EnvVar().
			WithName("WANDB_API_KEY").
			WithValueFrom(corev1apply.EnvVarSource().
				WithSecretKeyRef(corev1apply.SecretKeySelector().
					WithName(s.Name).
					WithKey(s.Key))))
	}

	podSpec := corev1apply.PodSpec().
		WithContainers(container).
		WithRestartPolicy(corev1.RestartPolicyNever)
	jobSpec := batchv1apply.JobSpec().
		WithTTLSecondsAfterFinished(int32(jobTTL.Seconds())).
		// Do not allow retries to simplify the failure handling.
		// TODO(kenji): Revisit.
		WithBackoffLimit(0).
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

func (p *JobClient) cmd(job *v1.Job, presult *PreProcessResult) (string, error) {
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
	additionalSFTArgs, err := toAddtionalSFTArgs(job)
	if err != nil {
		return "", err
	}

	params := Params{
		BaseModelName:     job.Model,
		BaseModelURLs:     presult.BaseModelURLs,
		TrainingFileURL:   presult.TrainingFileURL,
		ValidationFileURL: presult.ValidationFileURL,
		OutputModelURL:    presult.OutputModelURL,

		NumProcessors:     numProcessors,
		AdditionalSFTArgs: additionalSFTArgs,
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, &params); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (p *JobClient) getQueueName(namespace string) string {
	// TODO(aya): rethink how to get queue name
	return p.kueueConfig.DefaultQueueName
}

func (p *JobClient) cancelJob(ctx context.Context, ijob *v1.InternalJob) error {
	var kjob batchv1.Job
	if err := p.k8sClient.Get(ctx, types.NamespacedName{
		Name:      ijob.Job.Id,
		Namespace: ijob.Job.KubernetesNamespace,
	}, &kjob); err != nil {
		log := ctrl.LoggerFrom(ctx)
		log.V(2).Info("Failed to get the k8s job", "error", err)
		return client.IgnoreNotFound(err)
	}
	kjob.Spec.Suspend = ptr.To(true)
	return p.k8sClient.Update(ctx, &kjob, client.FieldOwner(jobManagerName))
}

func toAddtionalSFTArgs(job *v1.Job) (string, error) {
	args := []string{}
	if hp := job.Hyperparameters; hp != nil {

		if v := hp.BatchSize; v > 0 {
			args = append(args, fmt.Sprintf("--per_device_train_batch_size=%d", v))
		}
		if v := hp.LearningRateMultiplier; v > 0 {
			args = append(args, fmt.Sprintf("--learning_rate=%f", v))
		}
		if v := hp.NEpochs; v > 0 {
			args = append(args, fmt.Sprintf("--num_train_epochs=%d", v))
		}
	}

	if is := job.Integrations; len(is) > 0 {
		if len(is) > 1 {
			return "", fmt.Errorf("multiple integrations are not supported")
		}
		i := is[0]
		if i.Type != "wandb" {
			return "", fmt.Errorf("unsupported integration type: %s", i.Type)
		}
		w := i.Wandb
		if w == nil {
			return "", fmt.Errorf("wandb integration is not set")
		}
		args = append(args, "--report_to=wandb", fmt.Sprintf("--wandb_project=%s", w.Project))
	}

	return strings.Join(args, " "), nil

}

func isManagedJob(annotations map[string]string) bool {
	return annotations[managedJobAnnotationKey] == "true"
}
