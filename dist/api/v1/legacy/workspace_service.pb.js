/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/
import * as fm from "../../../fetch.pb";
export class WorkspaceWorkerService {
    static ListQueuedInternalNotebooks(req, initReq) {
        return fm.fetchReq(`/llmoperator.workspace.server.v1.WorkspaceWorkerService/ListQueuedInternalNotebooks`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
    static UpdateNotebookState(req, initReq) {
        return fm.fetchReq(`/llmoperator.workspace.server.v1.WorkspaceWorkerService/UpdateNotebookState`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
}
