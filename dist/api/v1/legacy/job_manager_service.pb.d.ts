import * as fm from "../../../fetch.pb";
import * as LlmarinerFine_tuningServerV1Job_manager_service from "../job_manager_service.pb";
export declare class FineTuningWorkerService {
    static ListQueuedInternalJobs(req: LlmarinerFine_tuningServerV1Job_manager_service.ListQueuedInternalJobsRequest, initReq?: fm.InitReq): Promise<LlmarinerFine_tuningServerV1Job_manager_service.ListQueuedInternalJobsResponse>;
    static GetInternalJob(req: LlmarinerFine_tuningServerV1Job_manager_service.GetInternalJobRequest, initReq?: fm.InitReq): Promise<LlmarinerFine_tuningServerV1Job_manager_service.InternalJob>;
    static UpdateJobPhase(req: LlmarinerFine_tuningServerV1Job_manager_service.UpdateJobPhaseRequest, initReq?: fm.InitReq): Promise<LlmarinerFine_tuningServerV1Job_manager_service.UpdateJobPhaseResponse>;
}
