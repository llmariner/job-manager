package server

import (
	"context"
	"errors"
	"time"

	"github.com/llm-operator/common/pkg/id"
	fv1 "github.com/llm-operator/file-manager/api/v1"
	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	mv1 "github.com/llm-operator/model-manager/api/v1"
	"github.com/llm-operator/rbac-manager/pkg/auth"
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
	userInfo, err := s.extractUserInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}

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

	if hp := req.Hyperparameters; hp != nil {
		if hp.BatchSize < 0 {
			return nil, status.Errorf(codes.InvalidArgument, "batch size must be non-negative")
		}
		if hp.LearningRateMultiplier < 0.0 {
			return nil, status.Errorf(codes.InvalidArgument, "learning rate multiplier must be non-negative")
		}
		if hp.NEpochs < 0 {
			return nil, status.Errorf(codes.InvalidArgument, "n epoch must be non-negative")
		}
	}

	// Pass the Authorization to the context for downstream gRPC calls.
	ctx = auth.CarryMetadata(ctx)
	if _, err := s.modelClient.GetModel(ctx, &mv1.GetModelRequest{
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

	jobID, err := id.GenerateIDForK8SResource("ftjob-")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate job id: %s", err)
	}

	var hp *v1.Job_Hyperparameters
	if rhp := req.Hyperparameters; rhp != nil {
		hp = &v1.Job_Hyperparameters{
			BatchSize:              rhp.BatchSize,
			LearningRateMultiplier: rhp.LearningRateMultiplier,
			NEpochs:                rhp.NEpochs,
		}
	}

	jobProto := &v1.Job{
		Id:              jobID,
		CreatedAt:       time.Now().UTC().Unix(),
		Model:           req.Model,
		TrainingFile:    req.TrainingFile,
		ValidationFile:  req.ValidationFile,
		Hyperparameters: hp,
		Object:          "fine_tuning.job",
		Status:          string(store.JobStateQueued),
		OrganizationId:  userInfo.OrganizationID,

		ProjectId:           userInfo.ProjectID,
		KubernetesNamespace: userInfo.KubernetesNamespace,
	}
	msg, err := proto.Marshal(jobProto)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal job: %s", err)
	}

	job := &store.Job{
		JobID:               jobID,
		State:               store.JobStateQueued,
		Message:             msg,
		Suffix:              req.Suffix,
		TenantID:            fakeTenantID,
		OrganizationID:      userInfo.OrganizationID,
		ProjectID:           userInfo.ProjectID,
		KubernetesNamespace: userInfo.KubernetesNamespace,
	}
	if err := s.store.CreateJob(job); err != nil {
		return nil, status.Errorf(codes.Internal, "create job: %s", err)
	}

	return jobProto, nil
}

func (s *S) validateFile(ctx context.Context, fileID string) error {
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
	userInfo, err := s.extractUserInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Limit < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "limit must be non-negative")
	}
	limit := req.Limit
	if limit == 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}

	var afterID uint
	if req.After != "" {
		job, err := s.store.GetJobByJobIDAndProjectID(req.After, userInfo.ProjectID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, status.Errorf(codes.InvalidArgument, "invalid after: %s", err)
			}
			return nil, status.Errorf(codes.Internal, "get job: %s", err)
		}
		afterID = job.ID
	}

	jobs, hasMore, err := s.store.ListJobsByProjectIDWithPagination(userInfo.ProjectID, afterID, int(limit))
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
		HasMore: hasMore,
	}, nil
}

// GetJob gets a job.
func (s *S) GetJob(
	ctx context.Context,
	req *v1.GetJobRequest,
) (*v1.Job, error) {
	userInfo, err := s.extractUserInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	// TODO(kenji): Check if a job is visible for a organization/project in the context.

	job, err := s.store.GetJobByJobIDAndProjectID(req.Id, userInfo.ProjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "get job: %s", err)
		}
		return nil, status.Errorf(codes.Internal, "get job: %s", err)
	}

	jobProto, err := job.V1Job()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "convert job to proto: %s", err)
	}
	return jobProto, nil
}

// CancelJob cancels a job.
func (s *S) CancelJob(
	ctx context.Context,
	req *v1.CancelJobRequest,
) (*v1.Job, error) {
	userInfo, err := s.extractUserInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	// TODO(kenji): Check if a job is visible for a organization/project in the context.

	job, err := s.store.GetJobByJobIDAndProjectID(req.Id, userInfo.ProjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "get job: %s", err)
		}
		return nil, status.Errorf(codes.Internal, "get job: %s", err)
	}

	jobProto, err := job.V1Job()
	if err != nil {
		return nil, err
	}
	switch job.State {
	case
		store.JobStateSucceeded,
		store.JobStatusFailed,
		store.JobStateCancelled:
		return jobProto, nil
	case store.JobStateQueued:
	case store.JobStateRunning:
		if err := s.k8sJobClient.CancelJob(ctx, jobProto, job.KubernetesNamespace); err != nil {
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

	jobProto, err = job.V1Job()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "convert job to proto: %s", err)
	}
	return jobProto, nil
}
