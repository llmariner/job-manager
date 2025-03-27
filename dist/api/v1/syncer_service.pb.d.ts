import * as fm from "../../fetch.pb";
export type PatchKubernetesObjectRequestResources = {
    gpu_limit?: number;
};
export type PatchKubernetesObjectRequest = {
    namespace?: string;
    name?: string;
    Group?: string;
    Version?: string;
    Resource?: string;
    resources?: PatchKubernetesObjectRequestResources;
    data?: Uint8Array;
};
export type PatchKubernetesObjectResponse = {
    cluster_id?: string;
    uid?: string;
};
export type DeleteKubernetesObjectRequest = {
    cluster_id?: string;
    namespace?: string;
    name?: string;
    Group?: string;
    Version?: string;
    Resource?: string;
};
export type DeleteKubernetesObjectResponse = {};
export type ListClusterIDsRequest = {};
export type ListClusterIDsResponse = {
    ids?: string[];
};
export declare class SyncerService {
    static PatchKubernetesObject(req: PatchKubernetesObjectRequest, initReq?: fm.InitReq): Promise<PatchKubernetesObjectResponse>;
    static DeleteKubernetesObject(req: DeleteKubernetesObjectRequest, initReq?: fm.InitReq): Promise<DeleteKubernetesObjectResponse>;
    static ListClusterIDs(req: ListClusterIDsRequest, initReq?: fm.InitReq): Promise<ListClusterIDsResponse>;
}
