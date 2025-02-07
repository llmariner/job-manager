/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../../fetch.pb"
export type PatchKubernetesObjectRequestResources = {
  gpuLimit?: number
}

export type PatchKubernetesObjectRequest = {
  namespace?: string
  name?: string
  group?: string
  version?: string
  resource?: string
  resources?: PatchKubernetesObjectRequestResources
  data?: Uint8Array
}

export type PatchKubernetesObjectResponse = {
  clusterId?: string
  uid?: string
}

export type DeleteKubernetesObjectRequest = {
  clusterId?: string
  namespace?: string
  name?: string
  group?: string
  version?: string
  resource?: string
}

export type DeleteKubernetesObjectResponse = {
}

export type ListClusterIDsRequest = {
}

export type ListClusterIDsResponse = {
  ids?: string[]
}

export class SyncerService {
  static PatchKubernetesObject(req: PatchKubernetesObjectRequest, initReq?: fm.InitReq): Promise<PatchKubernetesObjectResponse> {
    return fm.fetchReq<PatchKubernetesObjectRequest, PatchKubernetesObjectResponse>(`/llmariner.syncer.server.v1.SyncerService/PatchKubernetesObject`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static DeleteKubernetesObject(req: DeleteKubernetesObjectRequest, initReq?: fm.InitReq): Promise<DeleteKubernetesObjectResponse> {
    return fm.fetchReq<DeleteKubernetesObjectRequest, DeleteKubernetesObjectResponse>(`/llmariner.syncer.server.v1.SyncerService/DeleteKubernetesObject`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static ListClusterIDs(req: ListClusterIDsRequest, initReq?: fm.InitReq): Promise<ListClusterIDsResponse> {
    return fm.fetchReq<ListClusterIDsRequest, ListClusterIDsResponse>(`/llmariner.syncer.server.v1.SyncerService/ListClusterIDs`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
}