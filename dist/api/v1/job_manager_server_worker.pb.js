/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/
import * as fm from "../../fetch.pb";
export class JobWorkerService {
    static UpdateClusterStatus(req, initReq) {
        return fm.fetchReq(`/llmariner.jobs.server.v1.JobWorkerService/UpdateClusterStatus`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
}
