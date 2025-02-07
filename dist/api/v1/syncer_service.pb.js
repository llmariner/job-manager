/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/
import * as fm from "../../fetch.pb";
export class SyncerService {
    static PatchKubernetesObject(req, initReq) {
        return fm.fetchReq(`/llmariner.syncer.server.v1.SyncerService/PatchKubernetesObject`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
    static DeleteKubernetesObject(req, initReq) {
        return fm.fetchReq(`/llmariner.syncer.server.v1.SyncerService/DeleteKubernetesObject`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
    static ListClusterIDs(req, initReq) {
        return fm.fetchReq(`/llmariner.syncer.server.v1.SyncerService/ListClusterIDs`, Object.assign(Object.assign({}, initReq), { method: "POST", body: JSON.stringify(req) }));
    }
}
