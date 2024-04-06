package dispatcher

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/llm-operator/job-manager/common/pkg/store"
	"github.com/llm-operator/job-manager/dispatcher/internal/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewPodCreator returns a new PodCreator.
func NewPodCreator(
	k8sClient kubernetes.Interface,
	namespace string,
	modelStoreConfig *config.ModelStoreConfig,
	useFakeJob bool,
) *PodCreator {
	return &PodCreator{
		k8sClient:        k8sClient,
		namespace:        namespace,
		modelStoreConfig: modelStoreConfig,
		useFakeJob:       useFakeJob,
	}
}

// PodCreator creates a pod for a job.
type PodCreator struct {
	k8sClient kubernetes.Interface
	// TODO(kenji): Be able to specify the namespace per tenant.
	namespace        string
	modelStoreConfig *config.ModelStoreConfig
	useFakeJob       bool
}

func (p *PodCreator) createPod(ctx context.Context, job *store.Job) error {
	// TODO(kenji): Create a real fine-tuning job. See https://github.com/llm-operator/job-manager/tree/main/build/experiments/fine-tuning.

	log.Printf("Creating a pod for job %s\n", job.JobID)
	podName := fmt.Sprintf("job-%s", job.JobID)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
		},
		Spec: p.podSpec(),
	}
	if _, err := p.k8sClient.CoreV1().Pods(p.namespace).Create(ctx, pod, metav1.CreateOptions{}); err != nil {
		if apierrors.IsAlreadyExists(err) {
			// TODO(kenji): Revisit this error handling.
			log.Printf("Pod %s already exists\n", job.JobID)
			return nil
		}
		return err
	}

	log.Printf("Waiting for the pod to complete\n")
	ticker := time.NewTicker(10 * time.Second)
	var completed bool
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			pod, err := p.k8sClient.CoreV1().Pods(p.namespace).Get(ctx, podName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if pod.Status.Phase == corev1.PodSucceeded {
				completed = true
				break
			}

			log.Printf("Pod is still running. Check later\n")
		}

		if completed {
			break
		}
	}

	log.Printf("Pod finished running\n")
	return nil
}

func (p *PodCreator) podSpec() corev1.PodSpec {
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
			Requests: corev1.ResourceList{
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

	return corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:            "main",
				Image:           image,
				ImagePullPolicy: "Never",
				Command:         []string{"/bin/bash", "-c", cmd},
				Resources:       res,
				VolumeMounts:    volumeMounts,
			},
		},
		Volumes:       volumes,
		RestartPolicy: corev1.RestartPolicyOnFailure,
	}
}
