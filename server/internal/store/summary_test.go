package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetJobSummaries(t *testing.T) {
	// Create a new test store using NetTest pattern
	st, teardown := NewTest(t)
	defer teardown()

	// Test parameters
	tenantID := "test-tenant"
	projectID := "test-project"
	clusterID := "test-cluster"
	interval := time.Hour

	// Create test data: jobs in different states
	jobs := []*Job{
		{
			JobID:     "job-1",
			ProjectID: projectID,
			TenantID:  tenantID,
			ClusterID: clusterID,
			State:     JobStateRunning,
		},
		{
			JobID:     "job-2",
			ProjectID: projectID,
			TenantID:  tenantID,
			ClusterID: clusterID,
			State:     JobStateSucceeded,
		},
		{
			JobID:     "job-3",
			ProjectID: projectID,
			TenantID:  tenantID,
			ClusterID: clusterID,
			State:     JobStateRunning,
		},
		{
			JobID:     "job-4",
			ProjectID: projectID,
			TenantID:  tenantID,
			ClusterID: clusterID,
			State:     JobStateFailed,
		},
		{
			JobID:     "job-5",
			ProjectID: "different-project",
			TenantID:  tenantID,
			ClusterID: clusterID,
			State:     JobStateRunning,
		},
		{
			JobID:     "job-6",
			ProjectID: projectID,
			TenantID:  "different-tenant",
			ClusterID: clusterID,
			State:     JobStateRunning,
		},
	}

	// Insert test data into the database
	for _, job := range jobs {
		err := st.db.Create(job).Error
		require.NoError(t, err)
	}

	startTime := time.Now().Add(-24 * time.Hour) // 24 hours ago
	endTime := time.Now()

	// Test cases
	testCases := []struct {
		name                 string
		tenantID             string
		expectResults        bool
		expectedCount        int
		expectedRunningCount int
	}{
		{
			name:                 "Successful query with results",
			tenantID:             tenantID,
			expectResults:        true,
			expectedCount:        5,
			expectedRunningCount: 3,
		},
		{
			name:          "Query with no matching tenant",
			tenantID:      "non-existent-tenant",
			expectResults: true,
			expectedCount: 0,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function
			summaries, err := st.GetJobSummaries(tc.tenantID, startTime, endTime, interval)

			// Check results
			require.NoError(t, err)

			if tc.expectResults {
				if tc.expectedCount > 0 {
					assert.NotEmpty(t, summaries)

					// Count the total jobs across all summaries
					totalCount := int64(0)
					totalRunning := int64(0)
					for _, summary := range summaries {
						totalCount += summary.Count
						if summary.JobState == string(JobStateRunning) {
							totalRunning += summary.Count
						}
					}
					assert.Equal(t, int64(tc.expectedCount), totalCount)
					assert.Equal(t, int64(tc.expectedRunningCount), totalRunning)

					// Verify tenant ID in results
					for _, summary := range summaries {
						assert.Equal(t, tc.tenantID, summary.TenantID)
						assert.Equal(t, "fine_tuning", summary.JobType)
					}
				} else {
					assert.Empty(t, summaries)
				}
			}
		})
	}
}

func TestGetBatchJobSummaries(t *testing.T) {
	// Create a new test store using NetTest pattern
	st, teardown := NewTest(t)
	defer teardown()

	// Test parameters
	tenantID := "test-tenant"
	projectID := "test-project"
	clusterID := "test-cluster"
	interval := time.Hour

	// Create test data: batch jobs in different states
	batchJobs := []*BatchJob{
		{
			JobID:     "batch-job-1",
			ProjectID: projectID,
			TenantID:  tenantID,
			ClusterID: clusterID,
			State:     BatchJobStateRunning,
		},
		{
			JobID:     "batch-job-2",
			ProjectID: projectID,
			TenantID:  tenantID,
			ClusterID: clusterID,
			State:     BatchJobStateSucceeded,
		},
		{
			JobID:     "batch-job-3",
			ProjectID: projectID,
			TenantID:  tenantID,
			ClusterID: clusterID,
			State:     BatchJobStateRunning,
		},
		{
			JobID:     "batch-job-4",
			ProjectID: projectID,
			TenantID:  "different-tenant",
			ClusterID: clusterID,
			State:     BatchJobStateRunning,
		},
	}

	// Insert test data into the database
	for _, job := range batchJobs {
		err := st.db.Create(job).Error
		require.NoError(t, err)
	}

	startTime := time.Now().Add(-24 * time.Hour) // 24 hours ago
	endTime := time.Now()

	// Test cases
	testCases := []struct {
		name                 string
		tenantID             string
		expectResults        bool
		expectedCount        int
		expectedRunningCount int
	}{
		{
			name:                 "Successful query with results",
			tenantID:             tenantID,
			expectResults:        true,
			expectedCount:        3, // All batch jobs with matching tenant ID
			expectedRunningCount: 2,
		},
		{
			name:          "Query with no matching tenant",
			tenantID:      "non-existent-tenant",
			expectResults: true,
			expectedCount: 0,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function
			summaries, err := st.GetBatchJobSummaries(tc.tenantID, startTime, endTime, interval)

			// Check results
			require.NoError(t, err)

			if tc.expectResults {
				if tc.expectedCount > 0 {
					assert.NotEmpty(t, summaries)

					// Count the total batch jobs across all summaries
					totalCount := int64(0)
					totalRunning := int64(0)
					for _, summary := range summaries {
						totalCount += summary.Count
						if summary.JobState == string(BatchJobStateRunning) {
							totalRunning += summary.Count
						}
					}
					assert.Equal(t, int64(tc.expectedCount), totalCount)
					assert.Equal(t, int64(tc.expectedRunningCount), totalRunning)

					// Verify tenant ID in results
					for _, summary := range summaries {
						assert.Equal(t, tc.tenantID, summary.TenantID)
						assert.Equal(t, "batch", summary.JobType)
					}
				} else {
					assert.Empty(t, summaries)
				}
			}
		})
	}
}

func TestGetNotebookSummaries(t *testing.T) {
	// Create a new test store using NetTest pattern
	st, teardown := NewTest(t)
	defer teardown()

	// Test parameters
	tenantID := "test-tenant"
	projectID := "test-project"
	clusterID := "test-cluster"
	interval := time.Hour

	// Create test data: notebooks in different states
	notebooks := []*Notebook{
		{
			NotebookID: "notebook-1",
			Name:       "notebook-1",
			ProjectID:  projectID,
			TenantID:   tenantID,
			ClusterID:  clusterID,
			State:      NotebookStateRunning,
		},
		{
			NotebookID: "notebook-2",
			Name:       "notebook-2",
			ProjectID:  projectID,
			TenantID:   tenantID,
			ClusterID:  clusterID,
			State:      NotebookStateStopped,
		},
		{
			NotebookID: "notebook-3",
			Name:       "notebook-3",
			ProjectID:  projectID,
			TenantID:   "different-tenant",
			ClusterID:  clusterID,
			State:      NotebookStateRunning,
		},
	}

	// Insert test data into the database
	for _, notebook := range notebooks {
		err := st.db.Create(notebook).Error
		require.NoError(t, err)
	}

	startTime := time.Now().Add(-24 * time.Hour) // 24 hours ago
	endTime := time.Now()

	// Test cases
	testCases := []struct {
		name                 string
		tenantID             string
		expectResults        bool
		expectedCount        int
		expectedRunningCount int
	}{
		{
			name:                 "Successful query with results",
			tenantID:             tenantID,
			expectResults:        true,
			expectedCount:        2, // All notebooks with matching tenant ID
			expectedRunningCount: 1,
		},
		{
			name:          "Query with no matching tenant",
			tenantID:      "non-existent-tenant",
			expectResults: true,
			expectedCount: 0,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function
			summaries, err := st.GetNotebookSummaries(tc.tenantID, startTime, endTime, interval)

			// Check results
			require.NoError(t, err)

			if tc.expectResults {
				if tc.expectedCount > 0 {
					assert.NotEmpty(t, summaries)

					// Count the total notebooks across all summaries
					totalCount := int64(0)
					totalRunning := int64(0)
					for _, summary := range summaries {
						totalCount += summary.Count
						if summary.JobState == string(NotebookStateRunning) {
							totalRunning += summary.Count
						}
					}
					assert.Equal(t, int64(tc.expectedCount), totalCount)
					assert.Equal(t, int64(tc.expectedRunningCount), totalRunning)

					// Verify tenant ID in results
					for _, summary := range summaries {
						assert.Equal(t, tc.tenantID, summary.TenantID)
						assert.Equal(t, "notebook", summary.JobType)
					}
				} else {
					assert.Empty(t, summaries)
				}
			}
		})
	}
}
