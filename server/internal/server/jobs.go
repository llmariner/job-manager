package server

import (
	"context"

	v1 "github.com/llm-operator/job-manager/api/v1"
)

// CreateJob creates a new job.
func (s *S) CreateJob(
	ctx context.Context,
	req *v1.CreateJobRequest,
) (*v1.Job, error) {
	return &v1.Job{
		Id: "fake-id",
	}, nil
}

// ListJobs lists all jobs.
func (s *S) ListJobs(
	ctx context.Context,
	req *v1.ListJobRequest,
) (*v1.ListJobsResponse, error) {
	return &v1.ListJobsResponse{
		Object: "list",
		Data: []*v1.Job{
			{
				Id: "fake-id",
			},
		},
		HasMore: false,
	}, nil
}

// CancelJob cancels a job.
func (s *S) CancelJob(
	ctx context.Context,
	req *v1.CancelJobRequest,
) (*v1.Job, error) {
	return &v1.Job{
		Id: "fake-id",
	}, nil
}
