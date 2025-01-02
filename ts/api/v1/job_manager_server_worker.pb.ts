/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../../fetch.pb"
export type GpuNode = {
  resourceName?: string
  allocatableCount?: number
}

export type ProvisionableResource = {
  instanceFamily?: string
  instanceType?: string
}

export type ClusterStatus = {
  gpuNodes?: GpuNode[]
  provisionableResources?: ProvisionableResource[]
}

export type UpdateClusterStatusRequest = {
  clusterStatus?: ClusterStatus
}

export type UpdateClusterStatusResponse = {
}

export class JobWorkerService {
  static UpdateClusterStatus(req: UpdateClusterStatusRequest, initReq?: fm.InitReq): Promise<UpdateClusterStatusResponse> {
    return fm.fetchReq<UpdateClusterStatusRequest, UpdateClusterStatusResponse>(`/llmariner.jobs.server.v1.JobWorkerService/UpdateClusterStatus`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
}