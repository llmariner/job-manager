package dispatcher

import (
	"context"
	"fmt"

	"github.com/llm-operator/job-manager/common/pkg/store"
	"github.com/llm-operator/job-manager/dispatcher/internal/config"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	managedJobAnnotationKey = "llm-operator/managed-pod"
	jobIDAnnotationKey      = "llm-operator/job-id"

	jobPrefix = "job-"
)

// NewJobClient returns a new JobCreator.
func NewJobClient(
	k8sClient client.Client,
	namespace string,
	modelStoreConfig *config.ModelStoreConfig,
	useFakeJob bool,
	huggingFaceAccessToken string,
) *JobClient {
	return &JobClient{
		k8sClient:              k8sClient,
		namespace:              namespace,
		modelStoreConfig:       modelStoreConfig,
		useFakeJob:             useFakeJob,
		huggingFaceAccessToken: huggingFaceAccessToken,
	}
}

// JobClient operates a Kubernetes Job resource for a job.
type JobClient struct {
	k8sClient client.Client
	// TODO(kenji): Be able to specify the namespace per tenant.
	namespace              string
	modelStoreConfig       *config.ModelStoreConfig
	useFakeJob             bool
	huggingFaceAccessToken string
}

func (p *JobClient) createJob(ctx context.Context, jobData *store.Job) error {
	// TODO(kenji): Create a real fine-tuning job. See https://github.com/llm-operator/job-manager/tree/main/build/experiments/fine-tuning.
	log := ctrl.LoggerFrom(ctx)

	log.Info("Creating a pod for job")
	jobName := fmt.Sprintf("%s%s", jobPrefix, jobData.JobID)

	// TODO(kenji): Manage training files. Download them from the object store if needed.

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: p.namespace,
			Annotations: map[string]string{
				managedJobAnnotationKey: "true",
				jobIDAnnotationKey:      jobData.JobID,
			},
		},
		Spec: p.jobSpec(),
	}
	if err := p.k8sClient.Create(ctx, job); err != nil {
		if apierrors.IsAlreadyExists(err) {
			// TODO(kenji): Revisit this error handling.
			log.Info("Job already exists", "pod", fmt.Sprintf("%s/%s", job.Namespace, job.Name))
			return nil
		}
		return err
	}
	return nil
}

func (p *JobClient) jobSpec() batchv1.JobSpec {
	var image, cmd string
	var res corev1.ResourceRequirements
	if p.useFakeJob {
		image = "llm-operator/experiments-fake-job:latest"
		cmd = "mkdir /models/adapter; cp ./ggml-adapter-model.bin /models/adapter/ggml-adapter-model.bin"
	} else {
		image = "llm-operator/experiments-fine-tuning:latest"
		cmd = `
mkdir /models/adapter;
accelerate launch \
  --config_file=./single_gpu.yaml \
  --num_processes=1 \
  ./sft.py \
  --model_name=google/gemma-2b \
  --dataset_name=OpenAssistant/oasst_top1_2023-08-25 \
  --per_device_train_batch_size=2 \
  --gradient_accumulation_steps=1 \
  --max_steps=100 \
  --learning_rate=2e-4 \
  --save_steps=20_000 \
  --use_peft \
  --lora_r=16 \
  --lora_alpha=32 \
  --lora_target_modules q_proj k_proj v_proj o_proj \
  --load_in_4bit \
  --output_dir=./output &&
python ./convert-lora-to-ggml.py ./output &&
cp ./output/ggml-adapter-model.bin /models/adapter/
`
		res = corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"nvidia.com/gpu": *resource.NewQuantity(1, resource.DecimalSI),
			},
		}
	}

	var volumeMounts []corev1.VolumeMount
	var volumes []corev1.Volume
	mountPath := "/models"
	if ms := p.modelStoreConfig; ms.Enable {
		const vname = "model-store"
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      vname,
			MountPath: mountPath,
		})

		volumes = append(volumes, corev1.Volume{
			Name: vname,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: ms.PVClaimName,
				},
			},
		})
	}

	return batchv1.JobSpec{
		BackoffLimit:            ptr.To(int32(3)),
		TTLSecondsAfterFinished: ptr.To(int32(60 * 60 * 24)), // 1 day
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            "main",
						Image:           image,
						ImagePullPolicy: "Never",
						Command:         []string{"/bin/bash", "-c", cmd},
						Resources:       res,
						VolumeMounts:    volumeMounts,
						Env: []corev1.EnvVar{
							{
								Name:  "HUGGING_FACE_HUB_TOKEN",
								Value: p.huggingFaceAccessToken,
							},
						},
					},
				},
				Volumes:       volumes,
				RestartPolicy: corev1.RestartPolicyNever,
			},
		},
	}
}

func isManagedJob(annotations map[string]string) bool {
	return annotations[managedJobAnnotationKey] == "true"
}
