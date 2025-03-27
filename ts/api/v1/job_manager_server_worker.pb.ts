/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../../fetch.pb"
export type GpuNode = {
  resource_name?: string
  allocatable_count?: number
}

export type GpuPod = {
  resource_name?: string
  allocated_count?: number
  namespaced_name?: string
}

export type ProvisionableResource = {
  instance_family?: string
  instance_type?: string
}

export type ClusterStatus = {
  gpu_nodes?: GpuNode[]
  provisionable_resources?: ProvisionableResource[]
  gpu_pods?: GpuPod[]
}

export type UpdateClusterStatusRequest = {
  cluster_status?: ClusterStatus
}

export type UpdateClusterStatusResponse = {
}

export class JobWorkerService {
  static UpdateClusterStatus(req: UpdateClusterStatusRequest, initReq?: fm.InitReq): Promise<UpdateClusterStatusResponse> {
    return fm.fetchReq<UpdateClusterStatusRequest, UpdateClusterStatusResponse>(`/llmariner.jobs.server.v1.JobWorkerService/UpdateClusterStatus`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
}