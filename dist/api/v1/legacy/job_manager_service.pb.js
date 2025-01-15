/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/
import * as fm from "../../../fetch.pb";
export class FineTuningWorkerService {
    static ListQueuedInternalJobs(req, initReq) {
        return fm.fetchReq(`/llmoperator.fine_tuning.server.v1.FineTuningWorkerService/ListQueuedInternalJobs`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
    static GetInternalJob(req, initReq) {
        return fm.fetchReq(`/llmoperator.fine_tuning.server.v1.FineTuningWorkerService/GetInternalJob`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
    static UpdateJobPhase(req, initReq) {
        return fm.fetchReq(`/llmoperator.fine_tuning.server.v1.FineTuningWorkerService/UpdateJobPhase`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
}
