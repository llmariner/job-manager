package s3

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReconstructPresignURL(t *testing.T) {
	tcs := []struct {
		name   string
		url    string
		bucket string
		want   string
	}{
		{
			name:   "s3",
			url:    "https://s3-us-west-2.amazonaws.com",
			bucket: "mybucket",
			want:   "https://mybucket.s3-us-west-2.amazonaws.com",
		},
		{
			name:   "minio",
			url:    "http://localhost:9000",
			bucket: "mybucket",
			want:   "http://localhost:9000/mybucket",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, err := reconstructPresignURL(tc.url, tc.bucket)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
