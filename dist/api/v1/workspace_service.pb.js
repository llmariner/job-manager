/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/
import * as fm from "../../fetch.pb";
export var NotebookState;
(function (NotebookState) {
    NotebookState["STATE_UNSPECIFIED"] = "STATE_UNSPECIFIED";
    NotebookState["QUEUED"] = "QUEUED";
    NotebookState["INITIALIZING"] = "INITIALIZING";
    NotebookState["RUNNING"] = "RUNNING";
    NotebookState["STOPPED"] = "STOPPED";
    NotebookState["FAILED"] = "FAILED";
    NotebookState["DELETED"] = "DELETED";
})(NotebookState || (NotebookState = {}));
export var NotebookQueuedAction;
(function (NotebookQueuedAction) {
    NotebookQueuedAction["ACTION_UNSPECIFIED"] = "ACTION_UNSPECIFIED";
    NotebookQueuedAction["STARTING"] = "STARTING";
    NotebookQueuedAction["STOPPING"] = "STOPPING";
    NotebookQueuedAction["DELETING"] = "DELETING";
})(NotebookQueuedAction || (NotebookQueuedAction = {}));
export class WorkspaceService {
    static CreateNotebook(req, initReq) {
        return fm.fetchReq(`/v1/workspaces/notebooks`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
    static ListNotebooks(req, initReq) {
        return fm.fetchReq(`/v1/workspaces/notebooks?${fm.renderURLSearchParams(req, [])}`, Object.assign(Object.assign({}, initReq), { method: "GET" }));
    }
    static GetNotebook(req, initReq) {
        return fm.fetchReq(`/v1/workspaces/notebooks/${req["id"]}?${fm.renderURLSearchParams(req, ["id"])}`, Object.assign(Object.assign({}, initReq), { method: "GET" }));
    }
    static DeleteNotebook(req, initReq) {
        return fm.fetchReq(`/v1/workspaces/notebooks/${req["id"]}`, Object.assign(Object.assign({}, initReq), { method: "DELETE" }));
    }
    static StopNotebook(req, initReq) {
        return fm.fetchReq(`/v1/workspaces/notebooks/${req["id"]}/actions:stop`, Object.assign(Object.assign({}, initReq), { method: "POST" }));
    }
    static StartNotebook(req, initReq) {
        return fm.fetchReq(`/v1/workspaces/notebooks/${req["id"]}/actions:start`, Object.assign(Object.assign({}, initReq), { method: "POST" }));
    }
}
export class WorkspaceWorkerService {
    static ListQueuedInternalNotebooks(req, initReq) {
        return fm.fetchReq(`/llmariner.workspace.server.v1.WorkspaceWorkerService/ListQueuedInternalNotebooks`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
    static UpdateNotebookState(req, initReq) {
        return fm.fetchReq(`/llmariner.workspace.server.v1.WorkspaceWorkerService/UpdateNotebookState`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
}
