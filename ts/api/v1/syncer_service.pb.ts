/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../../fetch.pb"
export type PutKubernetesObjectRequestResources = {
  gpuLimit?: number
}

export type PutKubernetesObjectRequest = {
  namespace?: string
  name?: string
  group?: string
  version?: string
  resource?: string
  resources?: PutKubernetesObjectRequestResources
  data?: Uint8Array
}

export type PutKubernetesObjectResponse = {
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

export class SyncerService {
  static PutKubernetesObject(req: PutKubernetesObjectRequest, initReq?: fm.InitReq): Promise<PutKubernetesObjectResponse> {
    return fm.fetchReq<PutKubernetesObjectRequest, PutKubernetesObjectResponse>(`/llmariner.syncer.server.v1.SyncerService/PutKubernetesObject`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static DeleteKubernetesObject(req: DeleteKubernetesObjectRequest, initReq?: fm.InitReq): Promise<DeleteKubernetesObjectResponse> {
    return fm.fetchReq<DeleteKubernetesObjectRequest, DeleteKubernetesObjectResponse>(`/llmariner.syncer.server.v1.SyncerService/DeleteKubernetesObject`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
}