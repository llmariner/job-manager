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

export enum NotebookState {
  STATE_UNSPECIFIED = "STATE_UNSPECIFIED",
  QUEUED = "QUEUED",
  INITIALIZING = "INITIALIZING",
  RUNNING = "RUNNING",
  STOPPED = "STOPPED",
  FAILED = "FAILED",
  DELETED = "DELETED",
  REQUEUED = "REQUEUED",
}

export enum NotebookQueuedAction {
  ACTION_UNSPECIFIED = "ACTION_UNSPECIFIED",
  STARTING = "STARTING",
  STOPPING = "STOPPING",
  DELETING = "DELETING",
  REQUEUEING = "REQUEUEING",
}

export type NotebookError = {
  code?: string
  message?: string
}

export type Notebook = {
  id?: string
  name?: string
  created_at?: string
  started_at?: string
  stopped_at?: string
  image?: string
  resources?: Resources
  envs?: {[key: string]: string}
  error?: NotebookError
  status?: string
  token?: string
  project_id?: string
  organization_id?: string
  kubernetes_namespace?: string
  cluster_id?: string
  organization_title?: string
  project_title?: string
  cluster_name?: string
}

export type ResourcesQuantity = {
  requests?: number
  limits?: number
}

export type Resources = {
  cpu_milicore?: ResourcesQuantity
  memory_megabytes?: ResourcesQuantity
  storage_megabytes?: ResourcesQuantity
  gpu_count?: number
}


type BaseCreateNotebookRequestImage = {
}

export type CreateNotebookRequestImage = BaseCreateNotebookRequestImage
  & OneOf<{ type: string; uri: string }>

export type CreateNotebookRequest = {
  name?: string
  image?: CreateNotebookRequestImage
  resources?: Resources
  envs?: {[key: string]: string}
}

export type ListNotebooksRequest = {
  after?: string
  limit?: number
}

export type ListNotebooksResponse = {
  notebooks?: Notebook[]
  has_more?: boolean
}

export type GetNotebookRequest = {
  id?: string
}

export type DeleteNotebookRequest = {
  id?: string
}

export type DeleteNotebookResponse = {
}

export type StopNotebookRequest = {
  id?: string
}

export type StartNotebookRequest = {
  id?: string
}

export type InternalNotebook = {
  notebook?: Notebook
  state?: NotebookState
  queued_action?: NotebookQueuedAction
}

export type ListQueuedInternalNotebooksRequest = {
}

export type ListQueuedInternalNotebooksResponse = {
  notebooks?: InternalNotebook[]
}

export type UpdateNotebookStateRequest = {
  id?: string
  state?: NotebookState
}

export type UpdateNotebookStateResponse = {
}

export class WorkspaceService {
  static CreateNotebook(req: CreateNotebookRequest, initReq?: fm.InitReq): Promise<Notebook> {
    return fm.fetchReq<CreateNotebookRequest, Notebook>(`/v1/workspaces/notebooks`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static ListNotebooks(req: ListNotebooksRequest, initReq?: fm.InitReq): Promise<ListNotebooksResponse> {
    return fm.fetchReq<ListNotebooksRequest, ListNotebooksResponse>(`/v1/workspaces/notebooks?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static GetNotebook(req: GetNotebookRequest, initReq?: fm.InitReq): Promise<Notebook> {
    return fm.fetchReq<GetNotebookRequest, Notebook>(`/v1/workspaces/notebooks/${req["id"]}?${fm.renderURLSearchParams(req, ["id"])}`, {...initReq, method: "GET"})
  }
  static DeleteNotebook(req: DeleteNotebookRequest, initReq?: fm.InitReq): Promise<DeleteNotebookResponse> {
    return fm.fetchReq<DeleteNotebookRequest, DeleteNotebookResponse>(`/v1/workspaces/notebooks/${req["id"]}`, {...initReq, method: "DELETE"})
  }
  static StopNotebook(req: StopNotebookRequest, initReq?: fm.InitReq): Promise<Notebook> {
    return fm.fetchReq<StopNotebookRequest, Notebook>(`/v1/workspaces/notebooks/${req["id"]}/actions:stop`, {...initReq, method: "POST"})
  }
  static StartNotebook(req: StartNotebookRequest, initReq?: fm.InitReq): Promise<Notebook> {
    return fm.fetchReq<StartNotebookRequest, Notebook>(`/v1/workspaces/notebooks/${req["id"]}/actions:start`, {...initReq, method: "POST"})
  }
}
export class WorkspaceWorkerService {
  static ListQueuedInternalNotebooks(req: ListQueuedInternalNotebooksRequest, initReq?: fm.InitReq): Promise<ListQueuedInternalNotebooksResponse> {
    return fm.fetchReq<ListQueuedInternalNotebooksRequest, ListQueuedInternalNotebooksResponse>(`/llmariner.workspace.server.v1.WorkspaceWorkerService/ListQueuedInternalNotebooks`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static UpdateNotebookState(req: UpdateNotebookStateRequest, initReq?: fm.InitReq): Promise<UpdateNotebookStateResponse> {
    return fm.fetchReq<UpdateNotebookStateRequest, UpdateNotebookStateResponse>(`/llmariner.workspace.server.v1.WorkspaceWorkerService/UpdateNotebookState`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
}