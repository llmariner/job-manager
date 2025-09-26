package server

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr/testr"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/server/internal/store"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestGetSummariesByTimeRange(t *testing.T) {
	st, teardown := store.NewTest(t)
	defer teardown()

	tenantID := "test-tenant"
	projectID := "test-project"
	clusterID := "test-cluster"

	// Define job types for testing - used later for validation
	expectedJobTypes := []v1.JobType{
		v1.JobType_JOB_TYPE_FINE_TUNING,
		v1.JobType_JOB_TYPE_BATCH,
		v1.JobType_JOB_TYPE_NOTEBOOK,
	}

	// Create test data: jobs in different states
	jobs := []*store.Job{
		{
			JobID:     "job-1",
			ProjectID: projectID,
			TenantID:  tenantID,
			ClusterID: clusterID,
			State:     store.JobStateRunning,
		},
		{
			JobID:     "job-2",
			ProjectID: projectID,
			TenantID:  tenantID,
			ClusterID: clusterID,
			State:     store.JobStateSucceeded,
		},
	}

	// Create test data: batch jobs in different states
	batchJobs := []*store.BatchJob{
		{
			JobID:     "batch-job-1",
			ProjectID: projectID,
			TenantID:  tenantID,
			ClusterID: clusterID,
			State:     store.BatchJobStateRunning,
		},
		{
			JobID:     "batch-job-2",
			ProjectID: projectID,
			TenantID:  tenantID,
			ClusterID: clusterID,
			State:     store.BatchJobStateQueued,
		},
	}

	// Create test data: notebooks in different states
	notebooks := []*store.Notebook{
		{
			NotebookID: "notebook-1",
			Name:       "notebook-1",
			ProjectID:  projectID,
			TenantID:   tenantID,
			ClusterID:  clusterID,
			State:      store.NotebookStateRunning,
		},
		{
			NotebookID: "notebook-2",
			Name:       "notebook-2",
			ProjectID:  projectID,
			TenantID:   tenantID,
			ClusterID:  clusterID,
			State:      store.NotebookStateStopped,
		},
	}

	// Insert test data into the database
	seed := &store.SeedStore{
		Jobs:      jobs,
		BatchJobs: batchJobs,
		Notebooks: notebooks,
	}
	store.Seed(t, st, seed)

	startTime := time.Now().Add(-24 * time.Hour) // 24 hours ago
	endTime := time.Now()

	// Test cases
	testCases := []struct {
		name               string
		tenantID           string
		expectResults      bool
		expectedRunning    int
		expectedCreated    int
		expectedStopped    int
		expectedUnfinished int
		jobTypes           []v1.JobType
		interval           time.Duration
	}{
		{
			name:               "Successful query with results",
			tenantID:           tenantID,
			expectResults:      true,
			expectedRunning:    3,
			expectedCreated:    6,
			expectedStopped:    1,
			expectedUnfinished: 4,
			jobTypes:           expectedJobTypes,
			interval:           time.Hour,
		},
		{
			name:               "Successful query with results",
			tenantID:           tenantID,
			expectResults:      true,
			expectedRunning:    3,
			expectedCreated:    6,
			expectedStopped:    1,
			expectedUnfinished: 4,
			jobTypes:           expectedJobTypes,
			interval:           time.Minute,
		},
		{
			name:               "Query with no matching tenant",
			tenantID:           "non-existent-tenant",
			expectResults:      true,
			expectedRunning:    0,
			expectedCreated:    0,
			expectedStopped:    0,
			expectedUnfinished: 0,
			jobTypes:           nil,
			interval:           time.Hour,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srv := New(st, nil, nil, &noopK8sClientFactory{}, &fakeScheduler{}, &fakeCache{}, nil, map[string]string{"t0": "img0"}, testr.New(t), nil)

			datapoints, err := srv.getSummariesByTimeRange(tc.tenantID, startTime, endTime, tc.interval)
			assert.NoError(t, err)

			if tc.expectResults {
				if tc.expectedCreated > 0 {
					assert.NotEmpty(t, datapoints)

					// Count the total jobs created across all datapoints and types
					totalCreated := int64(0)
					totalRunning := int64(0)
					totalStopped := int64(0)
					totalUnfinished := int64(0)
					for _, datapoint := range datapoints {
						for _, value := range datapoint.Values {
							totalCreated += value.TotalCreated
							totalRunning += value.TotalRunning
							totalStopped += value.TotalStopped
							totalUnfinished += value.TotalUnfinished
						}
					}
					assert.Equal(t, int64(tc.expectedCreated), totalCreated)
					assert.Equal(t, int64(tc.expectedRunning), totalRunning)
					assert.Equal(t, int64(tc.expectedStopped), totalStopped)
					assert.Equal(t, int64(tc.expectedUnfinished), totalUnfinished)
				} else {
					assert.Empty(t, datapoints)
				}
			}
		})
	}
}

// TestListJobSummaries verifies that the ListJobSummaries endpoint correctly returns job summaries
// with the expected counts and job types
func TestListJobSummaries(t *testing.T) {
	// Create a new test store
	st, teardown := store.NewTest(t)
	defer teardown()

	tenantID := "default-tenant-id"
	projectID := "test-project"
	clusterID := "test-cluster"

	// Define job types for testing
	expectedJobTypes := []v1.JobType{
		v1.JobType_JOB_TYPE_FINE_TUNING,
		v1.JobType_JOB_TYPE_BATCH,
		v1.JobType_JOB_TYPE_NOTEBOOK,
	}

	// Create test data: jobs in different states
	jobs := []*store.Job{
		{
			JobID:     "job-1",
			ProjectID: projectID,
			TenantID:  tenantID,
			ClusterID: clusterID,
			State:     store.JobStateRunning,
		},
		{
			JobID:     "job-2",
			ProjectID: projectID,
			TenantID:  tenantID,
			ClusterID: clusterID,
			State:     store.JobStateSucceeded,
		},
	}

	// Create test data: batch jobs in different states
	batchJobs := []*store.BatchJob{
		{
			JobID:     "batch-job-1",
			ProjectID: projectID,
			TenantID:  tenantID,
			ClusterID: clusterID,
			State:     store.BatchJobStateRunning,
		},
		{
			JobID:     "batch-job-2",
			ProjectID: projectID,
			TenantID:  tenantID,
			ClusterID: clusterID,
			State:     store.BatchJobStateQueued,
		},
	}

	// Create test data: notebooks in different states
	notebooks := []*store.Notebook{
		{
			NotebookID: "notebook-1",
			Name:       "notebook-1",
			ProjectID:  projectID,
			TenantID:   tenantID,
			ClusterID:  clusterID,
			State:      store.NotebookStateRunning,
		},
		{
			NotebookID: "notebook-2",
			Name:       "notebook-2",
			ProjectID:  projectID,
			TenantID:   tenantID,
			ClusterID:  clusterID,
			State:      store.NotebookStateStopped,
		},
	}

	// Insert test data into the database
	seed := &store.SeedStore{
		Jobs:      jobs,
		BatchJobs: batchJobs,
		Notebooks: notebooks,
	}
	store.Seed(t, st, seed)

	// Create server with the test store
	srv := New(st, nil, nil, &noopK8sClientFactory{}, &fakeScheduler{}, &fakeCache{}, nil, map[string]string{"t0": "img0"}, testr.New(t), nil)

	// Test cases
	testCases := []struct {
		name               string
		filter             *v1.RequestFilter
		expectResults      bool
		expectedRunning    int
		expectedCreated    int
		expectedStopped    int
		expectedUnfinished int
		jobTypes           []v1.JobType
	}{
		{
			name: "Custom duration filter",
			filter: &v1.RequestFilter{
				StartTimestamp: time.Now().Add(-24 * time.Hour).Unix(),
				EndTimestamp:   time.Now().Add(time.Hour).Unix(),
				Duration:       durationpb.New(2 * time.Hour),
			},
			expectResults:      true,
			expectedRunning:    3,
			expectedCreated:    6,
			expectedStopped:    1,
			expectedUnfinished: 4,
			jobTypes:           expectedJobTypes,
		},
		{
			name: "Trunc to previous hour",
			filter: &v1.RequestFilter{
				StartTimestamp: time.Now().Add(-2 * time.Hour).Unix(),
				EndTimestamp:   time.Now().Add(-1 * time.Hour).Unix(),
			},
			expectResults: false,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create auth context
			ctx := fakeAuthInto(context.Background())

			// Call the ListJobSummaries method
			resp, err := srv.ListJobSummaries(ctx, &v1.ListJobSummariesRequest{
				Filter: tc.filter,
			})
			assert.NoError(t, err)
			assert.NotNil(t, resp)

			// Verify results
			if tc.expectResults {
				// There should be datapoints
				assert.NotEmpty(t, resp.Datapoints)

				// Count the total jobs across all datapoints and types
				totalCreated := int64(0)
				totalRunning := int64(0)
				totalStopped := int64(0)
				totalUnfinished := int64(0)

				// Track which job types we've seen
				seenJobTypes := make(map[v1.JobType]bool)

				for _, datapoint := range resp.Datapoints {
					for _, value := range datapoint.Values {
						totalCreated += value.TotalCreated
						totalRunning += value.TotalRunning
						totalStopped += value.TotalStopped
						totalUnfinished += value.TotalUnfinished
						seenJobTypes[value.JobType] = true
					}
				}

				// Verify expected counts
				assert.Equal(t, int64(tc.expectedCreated), totalCreated)
				assert.Equal(t, int64(tc.expectedRunning), totalRunning)
				assert.Equal(t, int64(tc.expectedStopped), totalStopped)
				assert.Equal(t, int64(tc.expectedUnfinished), totalUnfinished)

				// Verify job types
				if tc.jobTypes != nil {
					for _, jobType := range tc.jobTypes {
						assert.True(t, seenJobTypes[jobType], "Expected to see job type %v", jobType)
					}
				}
			} else {
				assert.Empty(t, resp.Datapoints)
			}
		})
	}
}
