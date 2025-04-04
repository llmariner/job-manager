/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../../fetch.pb"

export enum InternalJobState {
  STATE_UNSPECIFIED = "STATE_UNSPECIFIED",
  QUEUED = "QUEUED",
  RUNNING = "RUNNING",
  FAILED = "FAILED",
  SUCCEEDED = "SUCCEEDED",
  CANCELED = "CANCELED",
}

export enum InternalJobAction {
  ACTION_UNSPECIFIED = "ACTION_UNSPECIFIED",
  CREATING = "CREATING",
  CANCELING = "CANCELING",
}

export enum UpdateJobPhaseRequestPhase {
  PHASE_UNSPECIFIED = "PHASE_UNSPECIFIED",
  PREPROCESSED = "PREPROCESSED",
  JOB_CREATED = "JOB_CREATED",
  FINETUNED = "FINETUNED",
  FAILED = "FAILED",
  RECREATE = "RECREATE",
  CANCELED = "CANCELED",
}

export type IntegrationWandb = {
  project?: string
  name?: string
  entity?: string
  tags?: string[]
}

export type Integration = {
  type?: string
  wandb?: IntegrationWandb
}

export type JobError = {
  code?: string
  message?: string
  param?: string
}

export type JobHyperparameters = {
  batch_size?: number
  learning_rate_multiplier?: number
  n_epochs?: number
}

export type JobResources = {
  gpu_count?: number
}

export type Job = {
  id?: string
  created_at?: string
  error?: JobError
  fine_tuned_model?: string
  finished_at?: string
  hyperparameters?: JobHyperparameters
  model?: string
  object?: string
  organization_id?: string
  result_files?: string[]
  status?: string
  trained_tokens?: number
  training_file?: string
  validation_file?: string
  integrations?: Integration[]
  seed?: number
  project_id?: string
  kubernetes_namespace?: string
  cluster_id?: string
  organization_title?: string
  project_title?: string
  cluster_name?: string
  resources?: JobResources
}

export type CreateJobRequestHyperparameters = {
  batch_size?: number
  learning_rate_multiplier?: number
  n_epochs?: number
}

export type CreateJobRequest = {
  model?: string
  training_file?: string
  hyperparameters?: CreateJobRequestHyperparameters
  suffix?: string
  validation_file?: string
  integrations?: Integration[]
  seed?: number
  resources?: JobResources
}

export type ListJobsRequest = {
  after?: string
  limit?: number
}

export type ListJobsResponse = {
  object?: string
  data?: Job[]
  has_more?: boolean
  total_items?: number
}

export type GetJobRequest = {
  id?: string
}

export type CancelJobRequest = {
  id?: string
}

export type InternalJob = {
  job?: Job
  output_model_id?: string
  suffix?: string
  state?: InternalJobState
  queued_action?: InternalJobAction
}

export type ListQueuedInternalJobsRequest = {
}

export type ListQueuedInternalJobsResponse = {
  jobs?: InternalJob[]
}

export type GetInternalJobRequest = {
  id?: string
}

export type UpdateJobPhaseRequest = {
  id?: string
  phase?: UpdateJobPhaseRequestPhase
  message?: string
  model_id?: string
}

export type UpdateJobPhaseResponse = {
}

export class FineTuningService {
  static CreateJob(req: CreateJobRequest, initReq?: fm.InitReq): Promise<Job> {
    return fm.fetchReq<CreateJobRequest, Job>(`/v1/fine_tuning/jobs`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static ListJobs(req: ListJobsRequest, initReq?: fm.InitReq): Promise<ListJobsResponse> {
    return fm.fetchReq<ListJobsRequest, ListJobsResponse>(`/v1/fine_tuning/jobs?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static GetJob(req: GetJobRequest, initReq?: fm.InitReq): Promise<Job> {
    return fm.fetchReq<GetJobRequest, Job>(`/v1/fine_tuning/jobs/${req["id"]}?${fm.renderURLSearchParams(req, ["id"])}`, {...initReq, method: "GET"})
  }
  static CancelJob(req: CancelJobRequest, initReq?: fm.InitReq): Promise<Job> {
    return fm.fetchReq<CancelJobRequest, Job>(`/v1/fine_tuning/jobs/${req["id"]}/cancel`, {...initReq, method: "POST"})
  }
}
export class FineTuningWorkerService {
  static ListQueuedInternalJobs(req: ListQueuedInternalJobsRequest, initReq?: fm.InitReq): Promise<ListQueuedInternalJobsResponse> {
    return fm.fetchReq<ListQueuedInternalJobsRequest, ListQueuedInternalJobsResponse>(`/llmariner.fine_tuning.server.v1.FineTuningWorkerService/ListQueuedInternalJobs`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static GetInternalJob(req: GetInternalJobRequest, initReq?: fm.InitReq): Promise<InternalJob> {
    return fm.fetchReq<GetInternalJobRequest, InternalJob>(`/llmariner.fine_tuning.server.v1.FineTuningWorkerService/GetInternalJob`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static UpdateJobPhase(req: UpdateJobPhaseRequest, initReq?: fm.InitReq): Promise<UpdateJobPhaseResponse> {
    return fm.fetchReq<UpdateJobPhaseRequest, UpdateJobPhaseResponse>(`/llmariner.fine_tuning.server.v1.FineTuningWorkerService/UpdateJobPhase`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
}