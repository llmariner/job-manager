/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/
import * as fm from "../../../fetch.pb";
export class BatchWorkerService {
    static ListQueuedInternalBatchJobs(req, initReq) {
        return fm.fetchReq(`/llmoperator.batch.server.v1.BatchWorkerService/ListQueuedInternalBatchJobs`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
    static GetInternalBatchJob(req, initReq) {
        return fm.fetchReq(`/llmoperator.batch.server.v1.BatchWorkerService/GetInternalBatchJob`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
    static UpdateBatchJobState(req, initReq) {
        return fm.fetchReq(`/llmoperator.batch.server.v1.BatchWorkerService/UpdateBatchJobState`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
}
