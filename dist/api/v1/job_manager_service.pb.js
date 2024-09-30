/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/
import * as fm from "../../fetch.pb";
export var InternalJobState;
(function (InternalJobState) {
    InternalJobState["STATE_UNSPECIFIED"] = "STATE_UNSPECIFIED";
    InternalJobState["QUEUED"] = "QUEUED";
    InternalJobState["RUNNING"] = "RUNNING";
    InternalJobState["FAILED"] = "FAILED";
    InternalJobState["SUCCEEDED"] = "SUCCEEDED";
    InternalJobState["CANCELED"] = "CANCELED";
})(InternalJobState || (InternalJobState = {}));
export var InternalJobAction;
(function (InternalJobAction) {
    InternalJobAction["ACTION_UNSPECIFIED"] = "ACTION_UNSPECIFIED";
    InternalJobAction["CREATING"] = "CREATING";
    InternalJobAction["CANCELING"] = "CANCELING";
})(InternalJobAction || (InternalJobAction = {}));
export var UpdateJobPhaseRequestPhase;
(function (UpdateJobPhaseRequestPhase) {
    UpdateJobPhaseRequestPhase["PHASE_UNSPECIFIED"] = "PHASE_UNSPECIFIED";
    UpdateJobPhaseRequestPhase["PREPROCESSED"] = "PREPROCESSED";
    UpdateJobPhaseRequestPhase["JOB_CREATED"] = "JOB_CREATED";
    UpdateJobPhaseRequestPhase["FINETUNED"] = "FINETUNED";
    UpdateJobPhaseRequestPhase["FAILED"] = "FAILED";
    UpdateJobPhaseRequestPhase["RECREATE"] = "RECREATE";
    UpdateJobPhaseRequestPhase["CANCELED"] = "CANCELED";
})(UpdateJobPhaseRequestPhase || (UpdateJobPhaseRequestPhase = {}));
export class FineTuningService {
    static CreateJob(req, initReq) {
        return fm.fetchReq(`/v1/fine_tuning/jobs`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
    static ListJobs(req, initReq) {
        return fm.fetchReq(`/v1/fine_tuning/jobs?${fm.renderURLSearchParams(req, [])}`, Object.assign(Object.assign({}, initReq), { method: "GET" }));
    }
    static GetJob(req, initReq) {
        return fm.fetchReq(`/v1/fine_tuning/jobs/${req["id"]}?${fm.renderURLSearchParams(req, ["id"])}`, Object.assign(Object.assign({}, initReq), { method: "GET" }));
    }
    static CancelJob(req, initReq) {
        return fm.fetchReq(`/v1/fine_tuning/jobs/${req["id"]}/cancel`, Object.assign(Object.assign({}, initReq), { method: "POST" }));
    }
}
export class FineTuningWorkerService {
    static ListQueuedInternalJobs(req, initReq) {
        return fm.fetchReq(`/llmariner.fine_tuning.server.v1.FineTuningWorkerService/ListQueuedInternalJobs`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
    static GetInternalJob(req, initReq) {
        return fm.fetchReq(`/llmariner.fine_tuning.server.v1.FineTuningWorkerService/GetInternalJob`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
    static UpdateJobPhase(req, initReq) {
        return fm.fetchReq(`/llmariner.fine_tuning.server.v1.FineTuningWorkerService/UpdateJobPhase`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
}
