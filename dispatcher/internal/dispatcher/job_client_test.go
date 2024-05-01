package dispatcher

import (
	"os"
	"testing"

	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	"github.com/llm-operator/job-manager/dispatcher/internal/config"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestJobCmd(t *testing.T) {
	tcs := []struct {
		name       string
		useFakeJob bool
		job        *v1.Job
		goldenFile string
	}{
		{
			name:       "basic",
			useFakeJob: false,
			job: &v1.Job{
				Model: "model-id",
			},

			goldenFile: "testdata/command.basic.golden",
		},
		{
			name:       "hyperparamters",
			useFakeJob: false,
			job: &v1.Job{
				Model: "model-id",
				Hyperparameters: &v1.Job_Hyperparameters{
					BatchSize:              32,
					LearningRateMultiplier: 0.1,
					NEpochs:                10,
				},
			},
			goldenFile: "testdata/command.hyperparameters.golden",
		},
		{
			name:       "fake",
			useFakeJob: true,
			job: &v1.Job{
				Model: "model-id",
			},
			goldenFile: "testdata/command.use_fake.golden",
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

			jc := NewJobClient(kc, "default", config.JobConfig{}, tc.useFakeJob)

			b, err := proto.Marshal(tc.job)
			assert.NoError(t, err)

			job := &store.Job{
				JobID:   "job-id",
				Message: b,
			}
			presult := &PreProcessResult{
				BaseModelURLs: map[string]string{
					"config.json": "https://example.com/config.json",
				},
				TrainingFileURL:   "https://example.com/training-file",
				ValidationFileURL: "https://example.com/validation-file",
				OutputModelURL:    "https://example.com/output-model",
			}
			got, err := jc.cmd(job, presult)
			assert.NoError(t, err)

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
			name: "basic",
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
			name: "empty",
			job:  &v1.Job{},
			want: "",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := toAddtionalSFTArgs(tc.job)
			assert.Equal(t, tc.want, got)
		})
	}
}
