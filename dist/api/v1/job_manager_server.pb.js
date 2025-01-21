/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/
import * as fm from "../../fetch.pb";
export class JobService {
    static ListClusters(req, initReq) {
        return fm.fetchReq(`/llmariner.jobs.server.v1.JobService/ListClusters`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
}
