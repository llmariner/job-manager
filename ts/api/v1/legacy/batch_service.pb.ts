/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../../../fetch.pb"
import * as LlmarinerBatchServerV1Batch_service from "../batch_service.pb"
export class BatchWorkerService {
  static ListQueuedInternalBatchJobs(req: LlmarinerBatchServerV1Batch_service.ListQueuedInternalBatchJobsRequest, initReq?: fm.InitReq): Promise<LlmarinerBatchServerV1Batch_service.ListQueuedInternalBatchJobsResponse> {
    return fm.fetchReq<LlmarinerBatchServerV1Batch_service.ListQueuedInternalBatchJobsRequest, LlmarinerBatchServerV1Batch_service.ListQueuedInternalBatchJobsResponse>(`/llmoperator.batch.server.v1.BatchWorkerService/ListQueuedInternalBatchJobs`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static GetInternalBatchJob(req: LlmarinerBatchServerV1Batch_service.GetInternalBatchJobRequest, initReq?: fm.InitReq): Promise<LlmarinerBatchServerV1Batch_service.InternalBatchJob> {
    return fm.fetchReq<LlmarinerBatchServerV1Batch_service.GetInternalBatchJobRequest, LlmarinerBatchServerV1Batch_service.InternalBatchJob>(`/llmoperator.batch.server.v1.BatchWorkerService/GetInternalBatchJob`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static UpdateBatchJobState(req: LlmarinerBatchServerV1Batch_service.UpdateBatchJobStateRequest, initReq?: fm.InitReq): Promise<LlmarinerBatchServerV1Batch_service.UpdateBatchJobStateResponse> {
    return fm.fetchReq<LlmarinerBatchServerV1Batch_service.UpdateBatchJobStateRequest, LlmarinerBatchServerV1Batch_service.UpdateBatchJobStateResponse>(`/llmoperator.batch.server.v1.BatchWorkerService/UpdateBatchJobState`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
}