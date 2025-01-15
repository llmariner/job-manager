import * as fm from "../../fetch.pb";
export type GpuNode = {
    resourceName?: string;
    allocatableCount?: number;
};
export type ProvisionableResource = {
    instanceFamily?: string;
    instanceType?: string;
};
export type ClusterStatus = {
    gpuNodes?: GpuNode[];
    provisionableResources?: ProvisionableResource[];
};
export type UpdateClusterStatusRequest = {
    clusterStatus?: ClusterStatus;
};
export type UpdateClusterStatusResponse = {};
export declare class JobWorkerService {
    static UpdateClusterStatus(req: UpdateClusterStatusRequest, initReq?: fm.InitReq): Promise<UpdateClusterStatusResponse>;
}
