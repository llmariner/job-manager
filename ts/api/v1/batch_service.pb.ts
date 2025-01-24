/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../../fetch.pb"

type Absent<T, K extends keyof T> = { [k in Exclude<keyof T, K>]?: undefined };
type OneOf<T> =
  | { [k in keyof T]?: undefined }
  | (
    keyof T extends infer K ?
      (K extends string & keyof T ? { [k in K]: T[K] } & Absent<T, K>
        : never)
    : never);

export enum InternalBatchJobState {
  STATE_UNSPECIFIED = "STATE_UNSPECIFIED",
  QUEUED = "QUEUED",
  RUNNING = "RUNNING",
  SUCCEEDED = "SUCCEEDED",
  FAILED = "FAILED",
  CANCELED = "CANCELED",
  DELETED = "DELETED",
}

export enum InternalBatchJobAction {
  ACTION_UNSPECIFIED = "ACTION_UNSPECIFIED",
  CREATING = "CREATING",
  CANCELING = "CANCELING",
  DELETING = "DELETING",
}

export type BatchJobError = {
  code?: string
  message?: string
}

export type BatchJobResources = {
  gpuCount?: number
}


type BaseBatchJobKind = {
}

export type BatchJobKind = BaseBatchJobKind
  & OneOf<{ pytorch: PyTorchJob }>

export type BatchJob = {
  id?: string
  createdAt?: string
  finishedAt?: string
  error?: BatchJobError
  status?: string
  image?: string
  command?: string
  resources?: BatchJobResources
  envs?: {[key: string]: string}
  dataFiles?: string[]
  projectId?: string
  kubernetesNamespace?: string
  clusterId?: string
  kind?: BatchJobKind
  organizationTitle?: string
  projectTitle?: string
  clusterName?: string
}

export type PyTorchJob = {
  workerCount?: number
}

export type CreateBatchJobRequest = {
  image?: string
  command?: string
  scripts?: {[key: string]: Uint8Array}
  resources?: BatchJobResources
  envs?: {[key: string]: string}
  dataFiles?: string[]
  kind?: BatchJobKind
}

export type ListBatchJobsRequest = {
  after?: string
  limit?: number
}

export type ListBatchJobsResponse = {
  jobs?: BatchJob[]
  hasMore?: boolean
}

export type GetBatchJobRequest = {
  id?: string
}

export type CancelBatchJobRequest = {
  id?: string
}

export type DeleteBatchJobRequest = {
  id?: string
}

export type InternalBatchJob = {
  job?: BatchJob
  state?: InternalBatchJobState
  queuedAction?: InternalBatchJobAction
}

export type ListQueuedInternalBatchJobsRequest = {
}

export type ListQueuedInternalBatchJobsResponse = {
  jobs?: InternalBatchJob[]
}

export type GetInternalBatchJobRequest = {
  id?: string
}

export type UpdateBatchJobStateRequest = {
  id?: string
  state?: InternalBatchJobState
  reason?: string
  message?: string
}

export type UpdateBatchJobStateResponse = {
}

export class BatchService {
  static CreateBatchJob(req: CreateBatchJobRequest, initReq?: fm.InitReq): Promise<BatchJob> {
    return fm.fetchReq<CreateBatchJobRequest, BatchJob>(`/v1/batch/jobs`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static ListBatchJobs(req: ListBatchJobsRequest, initReq?: fm.InitReq): Promise<ListBatchJobsResponse> {
    return fm.fetchReq<ListBatchJobsRequest, ListBatchJobsResponse>(`/v1/batch/jobs?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static GetBatchJob(req: GetBatchJobRequest, initReq?: fm.InitReq): Promise<BatchJob> {
    return fm.fetchReq<GetBatchJobRequest, BatchJob>(`/v1/batch/jobs/${req["id"]}?${fm.renderURLSearchParams(req, ["id"])}`, {...initReq, method: "GET"})
  }
  static CancelBatchJob(req: CancelBatchJobRequest, initReq?: fm.InitReq): Promise<BatchJob> {
    return fm.fetchReq<CancelBatchJobRequest, BatchJob>(`/v1/batch/jobs/${req["id"]}/cancel`, {...initReq, method: "POST"})
  }
  static DeleteBatchJob(req: DeleteBatchJobRequest, initReq?: fm.InitReq): Promise<BatchJob> {
    return fm.fetchReq<DeleteBatchJobRequest, BatchJob>(`/v1/batch/jobs/${req["id"]}`, {...initReq, method: "DELETE"})
  }
}
export class BatchWorkerService {
  static ListQueuedInternalBatchJobs(req: ListQueuedInternalBatchJobsRequest, initReq?: fm.InitReq): Promise<ListQueuedInternalBatchJobsResponse> {
    return fm.fetchReq<ListQueuedInternalBatchJobsRequest, ListQueuedInternalBatchJobsResponse>(`/llmariner.batch.server.v1.BatchWorkerService/ListQueuedInternalBatchJobs`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static GetInternalBatchJob(req: GetInternalBatchJobRequest, initReq?: fm.InitReq): Promise<InternalBatchJob> {
    return fm.fetchReq<GetInternalBatchJobRequest, InternalBatchJob>(`/llmariner.batch.server.v1.BatchWorkerService/GetInternalBatchJob`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static UpdateBatchJobState(req: UpdateBatchJobStateRequest, initReq?: fm.InitReq): Promise<UpdateBatchJobStateResponse> {
    return fm.fetchReq<UpdateBatchJobStateRequest, UpdateBatchJobStateResponse>(`/llmariner.batch.server.v1.BatchWorkerService/UpdateBatchJobState`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
}