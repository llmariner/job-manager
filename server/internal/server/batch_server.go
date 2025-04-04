package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/llmariner/common/pkg/id"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/server/internal/store"
	"github.com/llmariner/rbac-manager/pkg/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// CreateBatchJob creates a batch job.
func (s *S) CreateBatchJob(ctx context.Context, req *v1.CreateBatchJobRequest) (*v1.BatchJob, error) {
	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "failed to extract user info from context")
	}

	if req.Image == "" {
		return nil, status.Error(codes.InvalidArgument, "image is required")
	}
	image, ok := s.batchJobImages[req.Image]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "invalid image: %s", req.Image)
	}

	if req.Command == "" {
		return nil, status.Error(codes.InvalidArgument, "command is required")
	}

	if len(req.Scripts) == 0 {
		return nil, status.Error(codes.InvalidArgument, "scripts are required")
	}

	if k := req.Kind; k != nil {
		switch t := k.Kind.(type) {
		case *v1.BatchJob_Kind_Pytorch:
			if t.Pytorch.WorkerCount == 0 {
				return nil, status.Error(codes.InvalidArgument, "worker count must be set")
			}
		default:
			return nil, status.Errorf(codes.InvalidArgument, "invalid kind: %T", t)
		}
	}

	// Pass the Authorization to the context for downstream gRPC calls.
	ctx = auth.CarryMetadata(ctx)
	for _, fileID := range req.DataFiles {
		if err := s.validateFile(ctx, fileID); err != nil {
			return nil, err
		}
	}

	jobID, err := id.GenerateIDForK8SResource("bj-")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate batch job id: %s", err)
	}

	var gpuCount int
	if r := req.Resources; r != nil {
		gpuCount = int(r.GpuCount)
	}
	sresult, err := s.scheduler.Schedule(userInfo, "", gpuCount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "schedule: %s", err)
	}
	if err := s.cache.AddAssumedPod(userInfo.TenantID, sresult.ClusterID,
		fmt.Sprintf("%s/%s", sresult.Namespace, jobID), gpuCount); err != nil {
		return nil, status.Errorf(codes.Internal, "add assumed pod: %s", err)
	}

	jobProto := &v1.BatchJob{
		Id:                  jobID,
		CreatedAt:           time.Now().UTC().Unix(),
		Status:              string(store.BatchJobStateQueued),
		Image:               image,
		Command:             req.Command,
		Resources:           req.Resources,
		Envs:                req.Envs,
		DataFiles:           req.DataFiles,
		ProjectId:           userInfo.ProjectID,
		KubernetesNamespace: sresult.Namespace,
		ClusterId:           sresult.ClusterID,
		Kind:                req.Kind,

		OrganizationTitle: userInfo.OrganizationTitle,
		ProjectTitle:      userInfo.ProjectTitle,
		ClusterName:       sresult.ClusterName,
	}
	msg, err := proto.Marshal(jobProto)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal batch job: %s", err)
	}

	apikey, err := auth.ExtractTokenFromContext(ctx)
	if err != nil {
		return nil, err
	}
	kclient, err := s.k8sClientFactory.NewClient(sresult.ClusterID, apikey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create k8s client: %s", err)
	}
	if err := kclient.CreateSecret(ctx, jobID, sresult.Namespace, map[string][]byte{
		"OPENAI_API_KEY": []byte(apikey),
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "create secret: %s", err)
	}
	if err := kclient.CreateConfigMap(ctx, jobID, sresult.Namespace, req.Scripts); err != nil {
		return nil, status.Errorf(codes.Internal, "create configmap for scripts: %s", err)
	}

	job := &store.BatchJob{
		JobID:        jobID,
		Message:      msg,
		State:        store.BatchJobStateQueued,
		QueuedAction: store.BatchJobQueuedActionCreate,
		TenantID:     userInfo.TenantID,
		ProjectID:    userInfo.ProjectID,
		ClusterID:    sresult.ClusterID,
	}
	if err := s.store.CreateBatchJob(job); err != nil {
		return nil, status.Errorf(codes.Internal, "create batch job: %s", err)
	}
	return jobProto, nil
}

// ListBatchJobs lists batch jobs.
func (s *S) ListBatchJobs(ctx context.Context, req *v1.ListBatchJobsRequest) (*v1.ListBatchJobsResponse, error) {
	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "failed to extract user info from context")
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
		job, err := s.store.GetActiveBatchJobByIDAndProjectID(req.After, userInfo.ProjectID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, status.Errorf(codes.InvalidArgument, "invalid after: %s", err)
			}
			return nil, status.Errorf(codes.Internal, "get batch job: %s", err)
		}
		afterID = job.ID
	}

	jobs, hasMore, err := s.store.ListActiveBatchJobsByProjectIDWithPagination(userInfo.ProjectID, afterID, int(limit))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list batch jobs: %s", err)
	}

	var jobProtos []*v1.BatchJob
	for _, job := range jobs {
		jobProto, err := job.V1BatchJob()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "convert batch job to proto: %s", err)
		}
		jobProtos = append(jobProtos, jobProto)
	}

	totalItems, err := s.store.CountActiveBatchJobsByProjectID(userInfo.ProjectID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "count batch jobs: %s", err)
	}

	return &v1.ListBatchJobsResponse{
		Jobs:       jobProtos,
		HasMore:    hasMore,
		TotalItems: int32(totalItems),
	}, nil
}

// GetBatchJob gets a batch job.
func (s *S) GetBatchJob(ctx context.Context, req *v1.GetBatchJobRequest) (*v1.BatchJob, error) {
	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "failed to extract user info from context")
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	job, err := s.store.GetActiveBatchJobByIDAndProjectID(req.Id, userInfo.ProjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "get batch job: %s", err)
		}
		return nil, status.Errorf(codes.Internal, "get batch job: %s", err)
	}

	jobProto, err := job.V1BatchJob()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "convert batch job to proto: %s", err)
	}
	return jobProto, nil
}

// DeleteBatchJob deletes a batch job.
func (s *S) DeleteBatchJob(ctx context.Context, req *v1.DeleteBatchJobRequest) (*v1.BatchJob, error) {
	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "failed to extract user info from context")
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	job, err := s.store.GetActiveBatchJobByIDAndProjectID(req.Id, userInfo.ProjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "get batch job: %s", err)
		}
		return nil, status.Errorf(codes.Internal, "get batch job: %s", err)
	}
	jobProto, err := job.V1BatchJob()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "convert batch job to proto: %s", err)
	}

	if job.State == store.BatchJobStateDeleted ||
		(job.State == store.BatchJobStateQueued && job.QueuedAction == store.BatchJobQueuedActionDelete) {
		return jobProto, nil
	}

	job, err = s.store.SetBatchJobQueuedAction(job.JobID, job.Version, store.BatchJobQueuedActionDelete)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "set batch job queued action: %s", err)
	}
	jobProto, err = job.V1BatchJob()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "convert batch job to proto: %s", err)
	}
	return jobProto, nil
}

// CancelBatchJob cancels a batch job.
func (s *S) CancelBatchJob(ctx context.Context, req *v1.CancelBatchJobRequest) (*v1.BatchJob, error) {
	userInfo, ok := auth.ExtractUserInfoFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "failed to extract user info from context")
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	job, err := s.store.GetActiveBatchJobByIDAndProjectID(req.Id, userInfo.ProjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "get batch job: %s", err)
		}
		return nil, status.Errorf(codes.Internal, "get batch job: %s", err)
	}

	jobProto, err := job.V1BatchJob()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "convert batch job to proto: %s", err)
	}

	switch job.State {
	case store.BatchJobStateFailed,
		store.BatchJobStateCanceled,
		store.BatchJobStateSucceeded,
		store.BatchJobStateDeleted:
		return jobProto, nil
	case store.BatchJobStateRunning:
	case store.BatchJobStateQueued:
		if job.QueuedAction == store.BatchJobQueuedActionCancel {
			return jobProto, nil
		}
	default:
		return nil, status.Errorf(codes.Internal, "unknown batch job state: %s", job.State)
	}

	job, err = s.store.SetBatchJobQueuedAction(job.JobID, job.Version, store.BatchJobQueuedActionCancel)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "set batch job queued action: %s", err)
	}
	jobProto, err = job.V1BatchJob()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "convert batch job to proto: %s", err)
	}
	return jobProto, nil
}

// ListQueuedInternalBatchJobs lists queued internal batch jobs.
func (ws *WS) ListQueuedInternalBatchJobs(ctx context.Context, req *v1.ListQueuedInternalBatchJobsRequest) (*v1.ListQueuedInternalBatchJobsResponse, error) {
	clusterInfo, err := ws.extractClusterInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}

	jobs, err := ws.store.ListQueuedBatchJobsByTenantIDAndClusterID(clusterInfo.TenantID, clusterInfo.ClusterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list queued jobs: %s", err)
	}

	var ijobs []*v1.InternalBatchJob
	for _, j := range jobs {
		ij, err := j.V1InternalBatchJob()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "convert job to proto: %s", err)
		}
		ijobs = append(ijobs, ij)
	}

	return &v1.ListQueuedInternalBatchJobsResponse{Jobs: ijobs}, nil
}

// GetInternalBatchJob gets an internal batch job.
func (ws *WS) GetInternalBatchJob(ctx context.Context, req *v1.GetInternalBatchJobRequest) (*v1.InternalBatchJob, error) {
	clusterInfo, err := ws.extractClusterInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	job, err := ws.store.GetBatchJobByID(req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "batch job not found: %s", err)
		}
		return nil, status.Errorf(codes.Internal, "get batch job: %s", err)
	}
	if job.TenantID != clusterInfo.TenantID {
		return nil, status.Error(codes.NotFound, "batch job not found")
	}

	jobProto, err := job.V1InternalBatchJob()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "convert job to proto: %s", err)
	}
	return jobProto, nil
}

// UpdateBatchJobState updates the state of a batch job.
func (ws *WS) UpdateBatchJobState(ctx context.Context, req *v1.UpdateBatchJobStateRequest) (*v1.UpdateBatchJobStateResponse, error) {
	clusterInfo, err := ws.extractClusterInfoFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	job, err := ws.store.GetBatchJobByID(req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "batch job not found: %s", err)
		}
		return nil, status.Errorf(codes.Internal, "get batch job: %s", err)
	}
	if job.TenantID != clusterInfo.TenantID {
		return nil, status.Error(codes.NotFound, "batch job not found")
	}

	storeState := convertBatchJobState(req.State)
	if job.State == storeState {
		// Already in the desired state.
		return &v1.UpdateBatchJobStateResponse{}, nil
	}

	switch req.State {
	case v1.InternalBatchJob_STATE_UNSPECIFIED:
		return nil, status.Error(codes.InvalidArgument, "state is required")
	case v1.InternalBatchJob_RUNNING:
		if job.State != store.BatchJobStateQueued && job.QueuedAction != store.BatchJobQueuedActionCreate {
			// Queued state is only available in the store object and does not exist in the proto object.
			return nil, status.Errorf(codes.FailedPrecondition, "job state is not creating: %s (%s)", job.State, job.QueuedAction)
		}
		if err := ws.store.SetBatchJobState(job.JobID, job.Version, storeState); err != nil {
			return nil, status.Errorf(codes.Internal, "set batch job state: %s", err)
		}
		return &v1.UpdateBatchJobStateResponse{}, nil
	case v1.InternalBatchJob_SUCCEEDED:
		if job.State != store.BatchJobStateRunning {
			// Queued state is only available in the store object and does not exist in the proto object.
			return nil, status.Errorf(codes.FailedPrecondition, "job state is not running: %s", job.State)
		}
		if err := job.MutateMessage(func(job *v1.BatchJob) {
			job.FinishedAt = time.Now().UTC().Unix()
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "mutate batch job: %s", err)
		}
	case v1.InternalBatchJob_CANCELED:
		if job.State != store.BatchJobStateQueued && job.QueuedAction != store.BatchJobQueuedActionCancel {
			// Queued state is only available in the store object and does not exist in the proto object.
			return nil, status.Errorf(codes.FailedPrecondition, "job state is not canceling: %s (%s)", job.State, job.QueuedAction)
		}
		if err := job.MutateMessage(func(job *v1.BatchJob) {
			job.FinishedAt = time.Now().UTC().Unix()
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "mutate batch job: %s", err)
		}
	case v1.InternalBatchJob_DELETED:
		if job.State != store.BatchJobStateQueued && job.QueuedAction != store.BatchJobQueuedActionDelete {
			// Queued state is only available in the store object and does not exist in the proto object.
			return nil, status.Errorf(codes.FailedPrecondition, "job state is not queued: %s (%s)", job.State, job.QueuedAction)
		}
		if err := job.MutateMessage(func(job *v1.BatchJob) {
			job.FinishedAt = time.Now().UTC().Unix()
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "mutate batch job: %s", err)
		}
	case v1.InternalBatchJob_FAILED:
		if err := job.MutateMessage(func(job *v1.BatchJob) {
			job.FinishedAt = time.Now().UTC().Unix()
			job.Error = &v1.BatchJob_Error{
				Code:    req.Reason,
				Message: req.Message,
			}
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "mutate batch job: %s", err)
		}
	case v1.InternalBatchJob_QUEUED:
		return nil, status.Errorf(codes.FailedPrecondition, "unexpected state: %s", req.State)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unknown state: %s", req.State)
	}

	if err := ws.store.SetNonQueuedBatchJobStateAndMessage(job.JobID, job.Version, storeState, job.Message); err != nil {
		return nil, status.Errorf(codes.Internal, "set batch job state and message: %s", err)
	}
	return &v1.UpdateBatchJobStateResponse{}, nil
}

func convertBatchJobState(s v1.InternalBatchJob_State) store.BatchJobState {
	return store.BatchJobState(strings.ToLower(s.String()))
}
