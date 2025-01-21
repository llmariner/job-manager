import * as fm from "../../../fetch.pb";
import * as LlmarinerWorkspaceServerV1Workspace_service from "../workspace_service.pb";
export declare class WorkspaceWorkerService {
    static ListQueuedInternalNotebooks(req: LlmarinerWorkspaceServerV1Workspace_service.ListQueuedInternalNotebooksRequest, initReq?: fm.InitReq): Promise<LlmarinerWorkspaceServerV1Workspace_service.ListQueuedInternalNotebooksResponse>;
    static UpdateNotebookState(req: LlmarinerWorkspaceServerV1Workspace_service.UpdateNotebookStateRequest, initReq?: fm.InitReq): Promise<LlmarinerWorkspaceServerV1Workspace_service.UpdateNotebookStateResponse>;
}
