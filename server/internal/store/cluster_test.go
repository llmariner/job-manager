package store

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestCreateOrUpdateCluster(t *testing.T) {
	st, teardown := NewTest(t)
	defer teardown()

	c := &Cluster{
		ClusterID: "cid0",
		TenantID:  "tid0",
		Status:    []byte("s0"),
	}
	err := st.CreateOrUpdateCluster(c)
	assert.NoError(t, err)

	got, err := st.GetClusterByID("cid0")
	assert.NoError(t, err)
	assert.Equal(t, []byte("s0"), got.Status)

	_, err = st.GetClusterByID("cid1")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))

	c.Status = []byte("s1")
	err = st.CreateOrUpdateCluster(c)
	assert.NoError(t, err)

	got, err = st.GetClusterByID("cid0")
	assert.NoError(t, err)
	assert.Equal(t, []byte("s1"), got.Status)
}
