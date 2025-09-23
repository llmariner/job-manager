/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../../fetch.pb"
import * as LlmarinerJobsServerV1Job_manager_server_worker from "./job_manager_server_worker.pb"

export enum JobType {
  JOB_TYPE_UNSPECIFIED = "JOB_TYPE_UNSPECIFIED",
  JOB_TYPE_BATCH = "JOB_TYPE_BATCH",
  JOB_TYPE_FINE_TUNING = "JOB_TYPE_FINE_TUNING",
  JOB_TYPE_NOTEBOOK = "JOB_TYPE_NOTEBOOK",
}

export type ClusterSummary = {
  gpu_capacity?: number
  gpu_allocated?: number
  gpu_pod_count?: number
}

export type Cluster = {
  id?: string
  name?: string
  status?: LlmarinerJobsServerV1Job_manager_server_worker.ClusterStatus
  summary?: ClusterSummary
  last_updated_at?: string
}

export type ListClustersRequest = {
}

export type ListClustersResponse = {
  clusters?: Cluster[]
}

export type RequestFilter = {
  start_timestamp?: string
  end_timestamp?: string
}

export type ListJobSummariesRequest = {
  filter?: RequestFilter
}

export type ListJobSummariesResponseValue = {
  job_type?: JobType
  total_created?: string
  total_completed?: string
  total_cancelled?: string
  total_failed?: string
  total_deleted?: string
  total_running?: string
}

export type ListJobSummariesResponseDatapoint = {
  timestamp?: string
  values?: ListJobSummariesResponseValue[]
}

export type ListJobSummariesResponse = {
  datapoints?: ListJobSummariesResponseDatapoint[]
}

export class JobService {
  static ListClusters(req: ListClustersRequest, initReq?: fm.InitReq): Promise<ListClustersResponse> {
    return fm.fetchReq<ListClustersRequest, ListClustersResponse>(`/v1/jobs/clusters?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static ListJobSummaries(req: ListJobSummariesRequest, initReq?: fm.InitReq): Promise<ListJobSummariesResponse> {
    return fm.fetchReq<ListJobSummariesRequest, ListJobSummariesResponse>(`/v1/jobs/summaries?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
}