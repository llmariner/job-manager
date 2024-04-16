package dispatcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

	jc := NewJobClient(kc, "default", nil, false, "")
	_, err := jc.cmd()
	assert.NoError(t, err)
}
