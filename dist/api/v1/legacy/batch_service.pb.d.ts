import * as fm from "../../../fetch.pb";
import * as LlmarinerBatchServerV1Batch_service from "../batch_service.pb";
export declare class BatchWorkerService {
    static ListQueuedInternalBatchJobs(req: LlmarinerBatchServerV1Batch_service.ListQueuedInternalBatchJobsRequest, initReq?: fm.InitReq): Promise<LlmarinerBatchServerV1Batch_service.ListQueuedInternalBatchJobsResponse>;
    static GetInternalBatchJob(req: LlmarinerBatchServerV1Batch_service.GetInternalBatchJobRequest, initReq?: fm.InitReq): Promise<LlmarinerBatchServerV1Batch_service.InternalBatchJob>;
    static UpdateBatchJobState(req: LlmarinerBatchServerV1Batch_service.UpdateBatchJobStateRequest, initReq?: fm.InitReq): Promise<LlmarinerBatchServerV1Batch_service.UpdateBatchJobStateResponse>;
}
