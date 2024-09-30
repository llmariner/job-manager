/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../../../fetch.pb"
import * as LlmarinerFine_tuningServerV1Job_manager_service from "../job_manager_service.pb"
export class FineTuningWorkerService {
  static ListQueuedInternalJobs(req: LlmarinerFine_tuningServerV1Job_manager_service.ListQueuedInternalJobsRequest, initReq?: fm.InitReq): Promise<LlmarinerFine_tuningServerV1Job_manager_service.ListQueuedInternalJobsResponse> {
    return fm.fetchReq<LlmarinerFine_tuningServerV1Job_manager_service.ListQueuedInternalJobsRequest, LlmarinerFine_tuningServerV1Job_manager_service.ListQueuedInternalJobsResponse>(`/llmoperator.fine_tuning.server.v1.FineTuningWorkerService/ListQueuedInternalJobs`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static GetInternalJob(req: LlmarinerFine_tuningServerV1Job_manager_service.GetInternalJobRequest, initReq?: fm.InitReq): Promise<LlmarinerFine_tuningServerV1Job_manager_service.InternalJob> {
    return fm.fetchReq<LlmarinerFine_tuningServerV1Job_manager_service.GetInternalJobRequest, LlmarinerFine_tuningServerV1Job_manager_service.InternalJob>(`/llmoperator.fine_tuning.server.v1.FineTuningWorkerService/GetInternalJob`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
  static UpdateJobPhase(req: LlmarinerFine_tuningServerV1Job_manager_service.UpdateJobPhaseRequest, initReq?: fm.InitReq): Promise<LlmarinerFine_tuningServerV1Job_manager_service.UpdateJobPhaseResponse> {
    return fm.fetchReq<LlmarinerFine_tuningServerV1Job_manager_service.UpdateJobPhaseRequest, LlmarinerFine_tuningServerV1Job_manager_service.UpdateJobPhaseResponse>(`/llmoperator.fine_tuning.server.v1.FineTuningWorkerService/UpdateJobPhase`, {...initReq, method: "POST", body: JSON.stringify(req)})
  }
}