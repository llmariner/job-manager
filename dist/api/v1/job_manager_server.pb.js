/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/
import * as fm from "../../fetch.pb";
export var JobType;
(function (JobType) {
    JobType["JOB_TYPE_UNSPECIFIED"] = "JOB_TYPE_UNSPECIFIED";
    JobType["JOB_TYPE_BATCH"] = "JOB_TYPE_BATCH";
    JobType["JOB_TYPE_FINE_TUNING"] = "JOB_TYPE_FINE_TUNING";
    JobType["JOB_TYPE_NOTEBOOK"] = "JOB_TYPE_NOTEBOOK";
})(JobType || (JobType = {}));
export class JobService {
    static ListClusters(req, initReq) {
        return fm.fetchReq(`/v1/jobs/clusters?${fm.renderURLSearchParams(req, [])}`, Object.assign(Object.assign({}, initReq), { method: "GET" }));
    }
    static ListJobSummaries(req, initReq) {
        return fm.fetchReq(`/v1/jobs/summaries?${fm.renderURLSearchParams(req, [])}`, Object.assign(Object.assign({}, initReq), { method: "GET" }));
    }
}
