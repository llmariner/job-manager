package server

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr/testr"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/server/internal/store"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestUpdateClusterStatus(t *testing.T) {
	st, tearDown := store.NewTest(t)
	defer tearDown()

	_, err := st.GetClusterByID(defaultClusterID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))

	srv := NewWorkerServiceServer(st, testr.New(t))
	req := &v1.UpdateClusterStatusRequest{
		Status: &v1.ClusterStatus{},
	}
	_, err = srv.UpdateClusterStatus(fakeAuthInto(context.Background()), req)
	assert.NoError(t, err)

	got, err := st.GetClusterByID(defaultClusterID)
	assert.NoError(t, err)
	assert.Equal(t, defaultClusterID, got.ClusterID)

}
