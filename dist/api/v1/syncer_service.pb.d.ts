import * as fm from "../../fetch.pb";
export type PatchKubernetesObjectRequestResources = {
    gpuLimit?: number;
};
export type PatchKubernetesObjectRequest = {
    namespace?: string;
    name?: string;
    group?: string;
    version?: string;
    resource?: string;
    resources?: PatchKubernetesObjectRequestResources;
    data?: Uint8Array;
};
export type PatchKubernetesObjectResponse = {
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
    static PatchKubernetesObject(req: PatchKubernetesObjectRequest, initReq?: fm.InitReq): Promise<PatchKubernetesObjectResponse>;
    static DeleteKubernetesObject(req: DeleteKubernetesObjectRequest, initReq?: fm.InitReq): Promise<DeleteKubernetesObjectResponse>;
}
