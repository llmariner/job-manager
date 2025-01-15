import * as fm from "../../fetch.pb";
export type PutKubernetesObjectRequestResources = {
    gpuLimit?: number;
};
export type PutKubernetesObjectRequest = {
    namespace?: string;
    name?: string;
    group?: string;
    version?: string;
    resource?: string;
    resources?: PutKubernetesObjectRequestResources;
    data?: Uint8Array;
};
export type PutKubernetesObjectResponse = {
    clusterId?: string;
    uid?: string;
};
export type DeleteKubernetesObjectRequest = {
    clusterId?: string;
    namespace?: string;
    name?: string;
    group?: string;
    version?: string;
    resource?: string;
};
export type DeleteKubernetesObjectResponse = {};
export declare class SyncerService {
    static PutKubernetesObject(req: PutKubernetesObjectRequest, initReq?: fm.InitReq): Promise<PutKubernetesObjectResponse>;
    static DeleteKubernetesObject(req: DeleteKubernetesObjectRequest, initReq?: fm.InitReq): Promise<DeleteKubernetesObjectResponse>;
}
