package dispatcher

import (
	"os"
	"testing"

	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestJobCmd(t *testing.T) {
	kc := fake.NewFakeClient(&batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-job-id0",
			Namespace: "default",
		},
	})

	jc := NewJobClient(kc, "default", false)

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
		BaseModelURLs: map[string]string{"path": "config.json"},
	}
	got, err := jc.cmd(job, presult)
	assert.NoError(t, err)

	want, err := os.ReadFile("testdata/command.golden")
	assert.NoError(t, err)
	assert.Equal(t, string(want), got)
}
