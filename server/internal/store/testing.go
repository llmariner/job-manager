package store

import (
	"testing"

	"github.com/llmariner/common/pkg/gormlib/testdb"
	"github.com/stretchr/testify/assert"
)

// SeedStore is the data to seed into the test store.
type SeedStore struct {
	Jobs      []*Job
	Notebooks []*Notebook
	BatchJobs []*BatchJob
}

// NewTest returns a new test store.
func NewTest(t *testing.T) (*S, func()) {
	db, tearDown := testdb.New(t)
	err := autoMigrate(db)
	assert.NoError(t, err)
	return New(db), tearDown
}

// Seed seeds the test store with the given data.
func Seed(t *testing.T, st *S, data *SeedStore) {
	if data == nil {
		return
	}
	if len(data.Jobs) > 0 {
		err := st.db.Create(&data.Jobs).Error
		assert.NoError(t, err)
	}
	if len(data.Notebooks) > 0 {
		err := st.db.Create(&data.Notebooks).Error
		assert.NoError(t, err)
	}
	if len(data.BatchJobs) > 0 {
		err := st.db.Create(&data.BatchJobs).Error
		assert.NoError(t, err)
	}
}
