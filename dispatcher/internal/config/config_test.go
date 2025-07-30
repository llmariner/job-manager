package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoteBookConfigValidate(t *testing.T) {
	tcs := []struct {
		name    string
		c       *NotebooksConfig
		wantErr bool
	}{
		{
			name: "valid config",
			c: &NotebooksConfig{
				LLMarinerBaseURL: "http://localhost:8080",
				EnablePVC:        true,
				StorageClassName: "standard",
				StorageSize:      "5G",
				MountPath:        "/mnt/notebooks",
			},
		},
		{
			name: "invalid storage size",
			c: &NotebooksConfig{
				LLMarinerBaseURL: "http://localhost:8080",
				EnablePVC:        true,
				StorageClassName: "standard",
				StorageSize:      "5gb",
				MountPath:        "/mnt/notebooks",
			},
			wantErr: true,
		},
		{
			name: "invalid mount path",
			c: &NotebooksConfig{
				LLMarinerBaseURL: "http://localhost:8080",
				EnablePVC:        true,
				StorageClassName: "standard",
				StorageSize:      "5G",
				MountPath:        "relative/path",
			},
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.c.validate()
			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}
