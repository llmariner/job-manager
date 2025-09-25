package server

import (
	"context"
	"sort"
	"time"

	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/server/internal/store"
	"github.com/llmariner/rbac-manager/pkg/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultInterval = time.Hour
	defaultDuration = 7 * 24 * time.Hour
)

// ListJobSummaries returns job summary data for visualization.
// It summarizes job statistics by job type and timestamp.
func (s *S) ListJobSummaries(ctx context.Context, req *v1.ListJobSummariesRequest) (*v1.ListJobSummariesResponse, error) {
	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "failed to extract user info from context")
	}

	startTime, endTime, err := getStartEndTime(req.Filter, time.Now(), defaultDuration)
	if err != nil {
		return nil, err
	}

	duration := endTime.Sub(startTime)
	if req.Filter != nil && req.Filter.Duration != nil {
		duration = req.Filter.Duration.AsDuration()
	}

	datapoints, err := s.getSummariesByTimeRange(userInfo.TenantID, startTime, endTime, duration)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get summaries by time range: %s", err)
	}

	return &v1.ListJobSummariesResponse{
		Datapoints: datapoints,
	}, nil
}

func (s *S) getSummariesByTimeRange(tenantID string, startTime, endTime time.Time, interval time.Duration) ([]*v1.ListJobSummariesResponse_Datapoint, error) {
	// Create a map to store all job statistics by timestamp, job type, and job state
	timestampMap := make(map[int64]map[string]map[string]int64)

	var summaries []*store.JobSummary

	// Fetch fine-tuning jobs
	ftJobs, err := s.store.GetJobSummaries(tenantID, startTime, endTime, interval)
	if err != nil {
		return nil, err
	}
	summaries = append(summaries, ftJobs...)

	// Fetch batch jobs
	batchJobs, err := s.store.GetBatchJobSummaries(tenantID, startTime, endTime, interval)
	if err != nil {
		return nil, err
	}
	summaries = append(summaries, batchJobs...)

	// Fetch notebooks
	notebooks, err := s.store.GetNotebookSummaries(tenantID, startTime, endTime, interval)
	if err != nil {
		return nil, err
	}
	summaries = append(summaries, notebooks...)

	for _, summary := range summaries {
		// Initialize maps if needed
		if _, exists := timestampMap[summary.Timestamp]; !exists {
			timestampMap[summary.Timestamp] = make(map[string]map[string]int64)
		}

		if _, exists := timestampMap[summary.Timestamp][summary.JobType]; !exists {
			timestampMap[summary.Timestamp][summary.JobType] = make(map[string]int64)
		}

		// Accumulate counts by state
		timestampMap[summary.Timestamp][summary.JobType][summary.JobState] += summary.Count
		// count the number of new jobs created within the time range.
		if summary.Timestamp >= startTime.UnixNano() && summary.Timestamp < endTime.UnixNano() {
			timestampMap[summary.Timestamp][summary.JobType]["created"] += summary.Count
		}
	}

	// Convert to response format
	var datapoints []*v1.ListJobSummariesResponse_Datapoint

	// Get sorted list of timestamps
	var timestamps []int64
	for ts := range timestampMap {
		timestamps = append(timestamps, ts)
	}
	sort.Slice(timestamps, func(i, j int) bool { return timestamps[i] < timestamps[j] })

	for _, ts := range timestamps {
		datapoint := &v1.ListJobSummariesResponse_Datapoint{
			Timestamp: ts,
		}

		// Convert each job type's statistics
		for jobType, states := range timestampMap[ts] {
			// Create a value for this job type
			value := &v1.ListJobSummariesResponse_Value{
				JobType: toJobType(jobType),
			}

			// Set counts based on the job state
			for state, count := range states {
				switch state {
				case "succeeded": // JobStateSucceeded, BatchJobStateSucceeded
					value.TotalCompleted += count
				case "failed": // JobStateFailed, BatchJobStateFailed, NotebookStateFailed
					value.TotalFailed += count
				case "canceled": // JobStateCanceled, BatchJobStateCanceled
					value.TotalCancelled += count
				case "stopped": // NotebookStateStopped
					value.TotalStopped += count
				case "running": // JobStateRunning, BatchJobStateRunning, NotebookStateRunning
					value.TotalRunning += count
				case "initializing": // NotebookStateInitializing (treated as running)
					value.TotalRunning += count
				case "deleted": // NotebookStateDeleted, BatchJobStateDeleted
					value.TotalDeleted += count
				case "requeued": // NotebookStateRequeued
					value.TotalQueued += count
				case "queued": // JobStateQueued, BatchJobStateQueued
					value.TotalQueued += count
				case "created": // Newly created jobs within the time range
					value.TotalCreated += count
				}
			}
			value.TotalUnfinished = value.TotalQueued + value.TotalRunning
			datapoint.Values = append(datapoint.Values, value)
		}

		datapoints = append(datapoints, datapoint)
	}

	return datapoints, nil
}

func toJobType(jobType string) v1.JobType {
	switch jobType {
	case "batch":
		return v1.JobType_JOB_TYPE_BATCH
	case "fine_tuning":
		return v1.JobType_JOB_TYPE_FINE_TUNING
	case "notebook":
		return v1.JobType_JOB_TYPE_NOTEBOOK
	default:
		return v1.JobType_JOB_TYPE_UNSPECIFIED
	}
}

func getStartEndTime(filter *v1.RequestFilter, now time.Time, duration time.Duration) (time.Time, time.Time, error) {
	if filter == nil {
		filter = &v1.RequestFilter{}
	}

	var (
		startTime time.Time
		endTime   time.Time
	)

	switch t := filter.EndTimestamp; {
	case t > 0:
		endTime = time.Unix(t, 0)
	case t == 0:
		// Set the endtime so that it includes the most recent hour after truncation.
		//
		// But we also don't want to advance if there is no datapoint reported from the agent in
		// the most recent hour. So we add half of the default interval to the current time.
		endTime = now.Add(defaultInterval / 2)
	default:
		return time.Time{}, time.Time{}, status.Errorf(codes.InvalidArgument, "endTimestamp must be a non-negative value")
	}
	endTime = endTime.Truncate(defaultInterval)

	switch t := filter.StartTimestamp; {
	case t > 0:
		startTime = time.Unix(t, 0)
	case t == 0:
		startTime = endTime.Add(-1 * duration)
	default:
		return time.Time{}, time.Time{}, status.Errorf(codes.InvalidArgument, "startTimestamp must be a non-negative value")
	}
	startTime = startTime.Truncate(defaultInterval)

	if !startTime.Before(endTime) {
		return time.Time{}, time.Time{}, status.Errorf(codes.InvalidArgument, "startTimestamp must be before endTimestamp")
	}

	return startTime, endTime, nil
}
