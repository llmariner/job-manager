/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../../../fetch.pb"
import * as LlmarinerWorkspaceServerV1Workspace_service from "../workspace_service.pb"
export class WorkspaceWorkerService {
  static ListQueuedInternalNotebooks(req: LlmarinerWorkspaceServerV1Workspace_service.ListQueuedInternalNotebooksRequest, initReq?: fm.InitReq): Promise<LlmarinerWorkspaceServerV1Workspace_service.ListQueuedInternalNotebooksResponse> {
    return fm.fetchReq<LlmarinerWorkspaceServerV1Workspace_service.ListQueuedInternalNotebooksRequest, LlmarinerWorkspaceServerV1Workspace_service.ListQueuedInternalNotebooksResponse>(`/llmoperator.workspace.server.v1.WorkspaceWorkerService/ListQueuedInternalNotebooks`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static UpdateNotebookState(req: LlmarinerWorkspaceServerV1Workspace_service.UpdateNotebookStateRequest, initReq?: fm.InitReq): Promise<LlmarinerWorkspaceServerV1Workspace_service.UpdateNotebookStateResponse> {
    return fm.fetchReq<LlmarinerWorkspaceServerV1Workspace_service.UpdateNotebookStateRequest, LlmarinerWorkspaceServerV1Workspace_service.UpdateNotebookStateResponse>(`/llmoperator.workspace.server.v1.WorkspaceWorkerService/UpdateNotebookState`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
}