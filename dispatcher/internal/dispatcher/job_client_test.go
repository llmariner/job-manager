package dispatcher

import (
	"os"
	"testing"

	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/dispatcher/internal/config"
	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestJobCmd(t *testing.T) {
	tcs := []struct {
		name        string
		jobConfig   config.JobConfig
		job         *v1.Job
		goldenFile  string
		expGPUCount int
	}{
		{
			name:      "basic",
			jobConfig: config.JobConfig{},
			job: &v1.Job{
				Model: "model-id",
				Resources: &v1.Job_Resources{
					GpuCount: 2,
				},
			},
			goldenFile:  "testdata/command.basic.golden",
			expGPUCount: 2,
		},
		{
			name:      "hyperparamters",
			jobConfig: config.JobConfig{},
			job: &v1.Job{
				Model: "model-id",
				Hyperparameters: &v1.Job_Hyperparameters{
					BatchSize:              32,
					LearningRateMultiplier: 0.1,
					NEpochs:                10,
				},
			},
			goldenFile:  "testdata/command.hyperparameters.golden",
			expGPUCount: 1,
		},
		{
			name: "multi-gpu",
			job: &v1.Job{
				Model: "model-id",
				Resources: &v1.Job_Resources{
					GpuCount: 4,
				},
			},
			goldenFile:  "testdata/command.multi-gpu.golden",
			expGPUCount: 4,
		},
		{
			name: "curl flags",
			jobConfig: config.JobConfig{
				CurlFlags: "--insecure",
			},
			job: &v1.Job{
				Model: "model-id",
				Resources: &v1.Job_Resources{
					GpuCount: 2,
				},
			},
			goldenFile:  "testdata/command.curl_flags.golden",
			expGPUCount: 2,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			kc := fake.NewFakeClient(&batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "job-job-id0",
					Namespace: "default",
				},
			})

			jc := NewJobClient(kc, tc.jobConfig, config.KueueConfig{}, config.WorkloadConfig{})

			presult := &PreProcessResult{
				BaseModelURLs: map[string]string{
					"config.json": "https://example.com/config.json",
				},
				TrainingFileURL:         "https://example.com/training-file",
				ValidationFileURL:       "https://example.com/validation-file",
				OutputModelURL:          "https://example.com/output-model",
				OutputModelPresignFlags: "-F 'key=value'",
				Method:                  "supervised",
			}
			got, gpuCount, err := jc.cmd(tc.job, presult)
			assert.NoError(t, err)
			assert.Equal(t, tc.expGPUCount, gpuCount)
			want, err := os.ReadFile(tc.goldenFile)
			assert.NoError(t, err)
			assert.Equal(t, string(want), got)
		})
	}
}

func TestToAddtionalSFTArgs(t *testing.T) {
	tcs := []struct {
		name string
		job  *v1.Job
		want string
	}{
		{
			name: "hyperparameters",
			job: &v1.Job{
				Hyperparameters: &v1.Job_Hyperparameters{
					BatchSize:              32,
					LearningRateMultiplier: 0.1,
					NEpochs:                10,
				},
			},
			want: "--per_device_train_batch_size=32 --learning_rate=0.100000 --num_train_epochs=10",
		},
		{
			name: "integration",
			job: &v1.Job{
				Integrations: []*v1.Integration{
					{
						Type: "wandb",
						Wandb: &v1.Integration_Wandb{
							Project: "my-project",
						},
					},
				},
			},
			want: "--report_to=wandb --wandb_project=my-project",
		},
		{
			name: "empty",
			job:  &v1.Job{},
			want: "",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, err := toAddtionalSFTArgs(tc.job, config.JobConfig{})
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
