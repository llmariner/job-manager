/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/
import * as fm from "../../fetch.pb";
export var InternalBatchJobState;
(function (InternalBatchJobState) {
    InternalBatchJobState["STATE_UNSPECIFIED"] = "STATE_UNSPECIFIED";
    InternalBatchJobState["QUEUED"] = "QUEUED";
    InternalBatchJobState["RUNNING"] = "RUNNING";
    InternalBatchJobState["SUCCEEDED"] = "SUCCEEDED";
    InternalBatchJobState["FAILED"] = "FAILED";
    InternalBatchJobState["CANCELED"] = "CANCELED";
    InternalBatchJobState["DELETED"] = "DELETED";
})(InternalBatchJobState || (InternalBatchJobState = {}));
export var InternalBatchJobAction;
(function (InternalBatchJobAction) {
    InternalBatchJobAction["ACTION_UNSPECIFIED"] = "ACTION_UNSPECIFIED";
    InternalBatchJobAction["CREATING"] = "CREATING";
    InternalBatchJobAction["CANCELING"] = "CANCELING";
    InternalBatchJobAction["DELETING"] = "DELETING";
})(InternalBatchJobAction || (InternalBatchJobAction = {}));
export class BatchService {
    static CreateBatchJob(req, initReq) {
        return fm.fetchReq(`/v1/batch/jobs`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
    static ListBatchJobs(req, initReq) {
        return fm.fetchReq(`/v1/batch/jobs?${fm.renderURLSearchParams(req, [])}`, Object.assign(Object.assign({}, initReq), { method: "GET" }));
    }
    static GetBatchJob(req, initReq) {
        return fm.fetchReq(`/v1/batch/jobs/${req["id"]}?${fm.renderURLSearchParams(req, ["id"])}`, Object.assign(Object.assign({}, initReq), { method: "GET" }));
    }
    static CancelBatchJob(req, initReq) {
        return fm.fetchReq(`/v1/batch/jobs/${req["id"]}/cancel`, Object.assign(Object.assign({}, initReq), { method: "POST" }));
    }
    static DeleteBatchJob(req, initReq) {
        return fm.fetchReq(`/v1/batch/jobs/${req["id"]}`, Object.assign(Object.assign({}, initReq), { method: "DELETE" }));
    }
}
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
