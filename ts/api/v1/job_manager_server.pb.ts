/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../../fetch.pb"
import * as LlmarinerJobsServerV1Job_manager_server_worker from "./job_manager_server_worker.pb"
export type Cluster = {
  id?: string
  status?: LlmarinerJobsServerV1Job_manager_server_worker.ClusterStatus
  lastUpdatedAt?: string
}

export type ListClustersRequest = {
}

export type ListClustersResponse = {
  clusters?: Cluster[]
}

export class JobService {
  static ListClusters(req: ListClustersRequest, initReq?: fm.InitReq): Promise<ListClustersResponse> {
    return fm.fetchReq<ListClustersRequest, ListClustersResponse>(`/llmariner.jobs.server.v1.JobService/ListClusters`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
}