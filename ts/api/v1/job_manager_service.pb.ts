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
  batchSize?: number
  learningRateMultiplier?: number
  nEpochs?: number
}

export type Job = {
  id?: string
  createdAt?: string
  error?: JobError
  fineTunedModel?: string
  finishedAt?: string
  hyperparameters?: JobHyperparameters
  model?: string
  object?: string
  organizationId?: string
  resultFiles?: string[]
  status?: string
  trainedTokens?: number
  trainingFile?: string
  validationFile?: string
  integrations?: Integration[]
  seed?: number
  projectId?: string
  kubernetesNamespace?: string
  clusterId?: string
  organizationTitle?: string
  projectTitle?: string
  clusterName?: string
}

export type CreateJobRequestHyperparameters = {
  batchSize?: number
  learningRateMultiplier?: number
  nEpochs?: number
}

export type CreateJobRequest = {
  model?: string
  trainingFile?: string
  hyperparameters?: CreateJobRequestHyperparameters
  suffix?: string
  validationFile?: string
  integrations?: Integration[]
  seed?: number
}

export type ListJobsRequest = {
  after?: string
  limit?: number
}

export type ListJobsResponse = {
  object?: string
  data?: Job[]
  hasMore?: boolean
}

export type GetJobRequest = {
  id?: string
}

export type CancelJobRequest = {
  id?: string
}

export type InternalJob = {
  job?: Job
  outputModelId?: string
  suffix?: string
  state?: InternalJobState
  queuedAction?: InternalJobAction
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
  modelId?: string
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