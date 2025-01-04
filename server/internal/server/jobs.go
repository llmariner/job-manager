package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/llmariner/common/pkg/id"
	fv1 "github.com/llmariner/file-manager/api/v1"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/server/internal/store"
	mv1 "github.com/llmariner/model-manager/api/v1"
	"github.com/llmariner/rbac-manager/pkg/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// CreateJob creates a new job.
func (s *S) CreateJob(
	ctx context.Context,
	req *v1.CreateJobRequest,
) (*v1.Job, error) {
	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to extract user info from context")
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
		return nil, status.Errorf(codes.InvalidArgument, "get model path: %s", err)
	}

	if err := s.validateFile(ctx, req.TrainingFile); err != nil {
		return nil, err
	}
	if f := req.ValidationFile; f != "" {
		if err := s.validateFile(ctx, f); err != nil {
			return nil, err
		}
	}

	for _, i := range req.Integrations {
		if i.Type != "wandb" {
			return nil, status.Errorf(codes.InvalidArgument, "unsupported integration type: %s", i.Type)
		}
		wandb := i.Wandb
		if wandb == nil {
			return nil, status.Errorf(codes.InvalidArgument, "wandb is required")
		}
		if wandb.Project == "" {
			return nil, status.Errorf(codes.InvalidArgument, "wandb project is required")
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

	sresult, err := s.scheduler.Schedule(userInfo)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "schedule: %s", err)
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
		Integrations:    req.Integrations,
		Seed:            req.Seed,

		ProjectId:           userInfo.ProjectID,
		KubernetesNamespace: sresult.Namespace,
		ClusterId:           sresult.ClusterID,
	}
	msg, err := proto.Marshal(jobProto)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal job: %s", err)
	}

	job := &store.Job{
		JobID:          jobID,
		State:          store.JobStateQueued,
		QueuedAction:   store.JobQueuedActionCreate,
		Message:        msg,
		Suffix:         req.Suffix,
		TenantID:       userInfo.TenantID,
		OrganizationID: userInfo.OrganizationID,
		ProjectID:      userInfo.ProjectID,
		ClusterID:      sresult.ClusterID,
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
	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to extract user info from context")
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
	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to extract user info from context")
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
	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to extract user info from context")
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

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
		store.JobStateFailed,
		store.JobStateCanceled:
		return jobProto, nil
	case store.JobStateRunning:
	case store.JobStateQueued:
		if job.QueuedAction == store.JobQueuedActionCancel {
			return jobProto, nil
		}
	default:
		return nil, status.Errorf(codes.Internal, "unexpected job state: %s", job.State)
	}

	job, err = s.store.UpdateJobState(
		req.Id,
		job.Version,
		store.JobStateQueued,
		store.JobQueuedActionCancel,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update job state: %s", err)
	}
	jobProto, err = job.V1Job()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "convert job to proto: %s", err)
	}
	return jobProto, nil
}

// ListQueuedInternalJobs lists all queued internal jobs for the specified tenant.
func (ws *WS) ListQueuedInternalJobs(ctx context.Context, req *v1.ListQueuedInternalJobsRequest) (resp *v1.ListQueuedInternalJobsResponse, err error) {
	clusterInfo, err := ws.extractClusterInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}

	jobs, err := ws.store.ListQueuedJobsByTenantIDAndClusterID(clusterInfo.TenantID, clusterInfo.ClusterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list queued jobs: %s", err)
	}

	var ijobs []*v1.InternalJob
	for _, job := range jobs {
		jobProto, err := job.V1InternalJob()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "convert job to internal job: %s", err)
		}
		ijobs = append(ijobs, jobProto)
	}
	return &v1.ListQueuedInternalJobsResponse{Jobs: ijobs}, nil
}

// GetInternalJob gets an internal job.
func (ws *WS) GetInternalJob(ctx context.Context, req *v1.GetInternalJobRequest) (resp *v1.InternalJob, err error) {
	clusterInfo, err := ws.extractClusterInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	job, err := ws.store.GetJobByJobID(req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "get job: %s", err)
		}
		return nil, status.Errorf(codes.Internal, "get job: %s", err)
	}
	if job.TenantID != clusterInfo.TenantID {
		return nil, status.Error(codes.NotFound, "job not found")
	}

	jobProto, err := job.V1InternalJob()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "convert job to internal job: %s", err)
	}
	return jobProto, nil
}

// UpdateJobPhase updates the job status depending on the given phase.
func (ws *WS) UpdateJobPhase(ctx context.Context, req *v1.UpdateJobPhaseRequest) (resp *v1.UpdateJobPhaseResponse, err error) {
	clusterInfo, err := ws.extractClusterInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	job, err := ws.store.GetJobByJobID(req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "get job: %s", err)
		}
		return nil, status.Errorf(codes.Internal, "get job: %s", err)
	}
	if job.TenantID != clusterInfo.TenantID {
		return nil, status.Error(codes.NotFound, "job not found")
	}

	switch req.Phase {
	case v1.UpdateJobPhaseRequest_PHASE_UNSPECIFIED:
		return nil, status.Error(codes.InvalidArgument, "phase is required")
	case v1.UpdateJobPhaseRequest_PREPROCESSED:
		if job.State != store.JobStateQueued {
			return nil, status.Errorf(codes.FailedPrecondition, "job state is not queued: %s", job.State)
		}
		if req.ModelId == "" {
			return nil, status.Error(codes.InvalidArgument, "model id is required for preprocessed phase")
		}
		if err := ws.store.UpdateOutputModelID(req.Id, job.Version, req.ModelId); err != nil {
			return nil, status.Errorf(codes.Internal, "update output model ID: %s", err)
		}
	case v1.UpdateJobPhaseRequest_JOB_CREATED:
		if job.State != store.JobStateQueued {
			return nil, status.Errorf(codes.FailedPrecondition, "job state is not queued: %s", job.State)
		}
		if _, err := ws.store.UpdateJobState(req.Id, job.Version, store.JobStateRunning, ""); err != nil {
			return nil, status.Errorf(codes.Internal, "update job state: %s", err)
		}
	case v1.UpdateJobPhaseRequest_FINETUNED:
		if job.State != store.JobStateRunning {
			return nil, status.Errorf(codes.FailedPrecondition, "job state is not running: %s", job.State)
		}
		if req.ModelId == "" {
			return nil, status.Error(codes.InvalidArgument, "model id is required for fine-tuned phase")
		}
		if err := job.MutateMessage(func(j *v1.Job) {
			j.FinishedAt = time.Now().UTC().Unix()
			j.FineTunedModel = req.ModelId
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "mutate message: %s", err)
		}
		if err := ws.store.UpdateJobStateAndMessage(req.Id, job.Version, store.JobStateSucceeded, job.Message); err != nil {
			return nil, status.Errorf(codes.Internal, "update job state: %s", err)
		}
	case v1.UpdateJobPhaseRequest_CANCELED:
		if err := job.MutateMessage(func(j *v1.Job) {
			j.FinishedAt = time.Now().UTC().Unix()
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "mutate message: %s", err)
		}
		if err := ws.store.UpdateJobStateAndMessage(req.Id, job.Version, store.JobStateCanceled, job.Message); err != nil {
			return nil, status.Errorf(codes.Internal, "update job state: %s", err)
		}
	case v1.UpdateJobPhaseRequest_FAILED:
		if err := job.MutateMessage(func(j *v1.Job) {
			j.FinishedAt = time.Now().UTC().Unix()
			j.Error = &v1.Job_Error{Message: req.Message}
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "mutate message: %s", err)
		}
		if err := ws.store.UpdateJobStateAndMessage(req.Id, job.Version, store.JobStateFailed, job.Message); err != nil {
			return nil, status.Errorf(codes.Internal, "update job state: %s", err)
		}
	case v1.UpdateJobPhaseRequest_RECREATE:
		if job.State != store.JobStateRunning {
			return nil, status.Errorf(codes.FailedPrecondition, "job state is not running: %s", job.State)
		}
		if _, err := ws.store.UpdateJobState(req.Id, job.Version, store.JobStateQueued, store.JobQueuedActionCreate); err != nil {
			return nil, status.Errorf(codes.Internal, "update job state: %s", err)
		}
	default:
		return nil, status.Errorf(codes.Internal, "unknown phase: %v", req.Phase)
	}
	return &v1.UpdateJobPhaseResponse{}, nil
}
