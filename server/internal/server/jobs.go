package server

import (
	"context"

	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	"github.com/oklog/ulid/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	fakeTenantID = "fake-tenant-id"
)

// CreateJob creates a new job.
func (s *S) CreateJob(
	ctx context.Context,
	req *v1.CreateJobRequest,
) (*v1.Job, error) {

	// TODO(kenji): Validate the request.
	jobID := newJobID()
	job := &store.Job{
		JobID:    jobID,
		State:    store.JobStatePending,
		TenantID: fakeTenantID,
		// TODO(kenji): Fill more field.
	}
	if err := s.store.CreateJob(job); err != nil {
		return nil, status.Errorf(codes.Internal, "create job: %s", err)
	}

	return &v1.Job{
		Id: jobID,
	}, nil
}

// ListJobs lists all jobs.
func (s *S) ListJobs(
	ctx context.Context,
	req *v1.ListJobRequest,
) (*v1.ListJobsResponse, error) {
	jobs, err := s.store.FindJobs(fakeTenantID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "find jobs: %s", err)
	}

	// TODO: Implement pagination.
	var jobProtos []*v1.Job
	for _, job := range jobs {
		jobProtos = append(jobProtos, &v1.Job{
			// TODO(kenji): Fill more field.
			Id: job.JobID,
		})
	}
	return &v1.ListJobsResponse{
		Object:  "list",
		Data:    jobProtos,
		HasMore: false,
	}, nil
}

// CancelJob cancels a job.
func (s *S) CancelJob(
	ctx context.Context,
	req *v1.CancelJobRequest,
) (*v1.Job, error) {
	return nil, status.Errorf(codes.Unimplemented, "not implemented")
}

func newJobID() string {
	return ulid.Make().String()
}
