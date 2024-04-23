package server

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	fv1 "github.com/llm-operator/file-manager/api/v1"
	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	mv1 "github.com/llm-operator/model-manager/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

const (
	fakeTenantID = "fake-tenant-id"
)

// CreateJob creates a new job.
func (s *S) CreateJob(
	ctx context.Context,
	req *v1.CreateJobRequest,
) (*v1.Job, error) {
	// TODO(kenji): Add more validation.
	if req.Model == "" {
		return nil, status.Errorf(codes.InvalidArgument, "model is required")
	}
	if req.TrainingFile == "" {
		return nil, status.Errorf(codes.InvalidArgument, "training file is required")
	}
	if req.Suffix == "" {
		return nil, status.Errorf(codes.InvalidArgument, "suffix is required")
	}
	// TODO(kenji): This follows the OpenAI API spec, but might not be necessary.
	if len(req.Suffix) > 18 {
		return nil, status.Errorf(codes.InvalidArgument, "suffix is too long")
	}

	if _, err := s.modelClient.GetBaseModelPath(ctx, &mv1.GetBaseModelPathRequest{
		Id: req.Model,
	}); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, status.Errorf(codes.InvalidArgument, "model not found")
		}
		return nil, status.Errorf(codes.InvalidArgument, "get base model path: %s", err)
	}

	if err := s.validateFile(ctx, req.TrainingFile); err != nil {
		return nil, err
	}
	if f := req.ValidationFile; f != "" {
		if err := s.validateFile(ctx, f); err != nil {
			return nil, err
		}
	}

	jobID := newJobID()

	jobProto := &v1.Job{
		Id:             jobID,
		CreatedAt:      time.Now().UTC().Unix(),
		Model:          req.Model,
		TrainingFile:   req.TrainingFile,
		ValidationFile: req.ValidationFile,
		Object:         "fine_tuning.job",
		Status:         string(store.JobStateQueued),
	}
	msg, err := proto.Marshal(jobProto)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal job: %s", err)
	}

	job := &store.Job{
		JobID:    jobID,
		State:    store.JobStateQueued,
		Message:  msg,
		Suffix:   req.Suffix,
		TenantID: fakeTenantID,
		// TODO(kenji): Fill more field.
	}
	if err := s.store.CreateJob(job); err != nil {
		return nil, status.Errorf(codes.Internal, "create job: %s", err)
	}

	return jobProto, nil
}

func (s *S) validateFile(ctx context.Context, fileID string) error {
	// Check if the specified training file exits.
	// TODO: Pass the authorization token.
	if _, err := s.fileGetClient.GetFile(ctx, &fv1.GetFileRequest{
		Id: fileID,
	}); err != nil {
		if status.Code(err) == codes.NotFound {
			return status.Errorf(codes.InvalidArgument, "file %q not found", fileID)
		}
		return status.Errorf(codes.InvalidArgument, "get file: %s", err)
	}
	return nil
}

// ListJobs lists all jobs.
func (s *S) ListJobs(
	ctx context.Context,
	req *v1.ListJobsRequest,
) (*v1.ListJobsResponse, error) {
	jobs, err := s.store.ListJobsByTenantID(fakeTenantID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "find jobs: %s", err)
	}

	// TODO: Implement pagination.
	var jobProtos []*v1.Job
	for _, job := range jobs {
		jobProto, err := job.V1Job()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "convert job to proto: %s", err)
		}
		jobProtos = append(jobProtos, jobProto)
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
	job, err := s.store.GetJobByJobID(req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "get job: %s", err)
		}
		return nil, status.Errorf(codes.Internal, "get job: %s", err)
	}

	switch job.State {
	case
		store.JobStateSucceeded,
		store.JobStatusFailed,
		store.JobStateCancelled:
		return job.V1Job()
	case store.JobStateQueued:
	case store.JobStateRunning:
		if err := s.k8sJobClient.CancelJob(ctx, job.JobID); err != nil {
			return nil, status.Errorf(codes.Internal, "cancel job: %s", err)
		}
	default:
		return nil, status.Errorf(codes.Internal, "unexpected job state: %s", job.State)
	}

	if err := job.MutateMessage(func(j *v1.Job) {
		j.FinishedAt = time.Now().UTC().Unix()
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "mutate message: %s", err)
	}
	if err := s.store.UpdateJobStateAndMessage(
		req.Id,
		job.Version,
		store.JobStateCancelled,
		job.Message,
	); err != nil {
		return nil, status.Errorf(codes.Internal, "update job state: %s", err)
	}
	job.State = store.JobStateCancelled

	jobProto, err := job.V1Job()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "convert job to proto: %s", err)
	}
	return jobProto, nil
}

func newJobID() string {
	return uuid.New().String()
}
