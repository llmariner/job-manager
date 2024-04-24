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
		goldenFile string
	}{
		{
			name:       "non-fake",
			useFakeJob: false,
			goldenFile: "testdata/command.golden",
		},
		{
			name:       "fake",
			useFakeJob: true,
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

			jobProto := &v1.Job{
				Model: "model-id",
			}
			b, err := proto.Marshal(jobProto)
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
