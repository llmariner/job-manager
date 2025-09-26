package store

import (
	"fmt"
	"time"
)

// JobSummary represents a summary record of jobs for a specific time period
type JobSummary struct {
	JobType   string
	JobState  string
	ProjectID string
	TenantID  string
	ClusterID string
	Timestamp int64
	Count     int64
}

// GetJobSummaries returns aggregated job statistics for a specific tenant within a time range
func (s *S) GetJobSummaries(tenantID string, startTime, endTime time.Time, interval time.Duration) ([]*JobSummary, error) {
	var summaries []*JobSummary

	// The timestamp is truncated to the specified interval
	truncExpr := fmt.Sprintf("created / %d * %d", interval.Nanoseconds(), interval.Nanoseconds())

	query := s.db.Model(&Job{}).
		Select(
			"'fine_tuning' AS job_type",
			"state AS job_state",
			"project_id",
			"tenant_id",
			"cluster_id",
			truncExpr+" AS timestamp",
			"COUNT(*) AS count",
		).
		Where("tenant_id = ?", tenantID).
		Where("updated_at >= ? AND updated_at < ?", startTime, endTime).
		Group("job_state, project_id, tenant_id, cluster_id, timestamp")

	if err := query.Scan(&summaries).Error; err != nil {
		return nil, fmt.Errorf("scan job summaries: %w", err)
	}

	return summaries, nil
}

// GetBatchJobSummaries returns aggregated batch job statistics for a specific tenant within a time range
func (s *S) GetBatchJobSummaries(tenantID string, startTime, endTime time.Time, interval time.Duration) ([]*JobSummary, error) {
	var summaries []*JobSummary

	// The timestamp is truncated to the specified interval
	truncExpr := fmt.Sprintf("created / %d * %d", interval.Nanoseconds(), interval.Nanoseconds())

	query := s.db.Model(&BatchJob{}).
		Select(
			"'batch' AS job_type",
			"state AS job_state",
			"project_id",
			"tenant_id",
			"cluster_id",
			truncExpr+" AS timestamp",
			"COUNT(*) AS count",
		).
		Where("tenant_id = ?", tenantID).
		Where("updated_at >= ? AND updated_at < ?", startTime, endTime).
		Group("job_state, project_id, tenant_id, cluster_id, timestamp")

	if err := query.Scan(&summaries).Error; err != nil {
		return nil, fmt.Errorf("scan batch job summaries: %w", err)
	}

	return summaries, nil
}

// GetNotebookSummaries returns aggregated notebook statistics for a specific tenant within a time range
func (s *S) GetNotebookSummaries(tenantID string, startTime, endTime time.Time, interval time.Duration) ([]*JobSummary, error) {
	var summaries []*JobSummary

	// The timestamp is truncated to the specified interval
	truncExpr := fmt.Sprintf("created / %d * %d", interval.Nanoseconds(), interval.Nanoseconds())

	query := s.db.Model(&Notebook{}).
		Select(
			"'notebook' AS job_type",
			"state AS job_state",
			"project_id",
			"tenant_id",
			"cluster_id",
			truncExpr+" AS timestamp",
			"COUNT(*) AS count",
		).
		Where("tenant_id = ?", tenantID).
		Where("updated_at >= ? AND updated_at < ?", startTime, endTime).
		Group("job_state, project_id, tenant_id, cluster_id, timestamp")

	if err := query.Scan(&summaries).Error; err != nil {
		return nil, fmt.Errorf("scan notebook summaries: %w", err)
	}

	return summaries, nil
}
