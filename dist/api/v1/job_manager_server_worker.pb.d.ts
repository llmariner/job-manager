import * as fm from "../../fetch.pb";
export type GpuNode = {
    resource_name?: string;
    allocatable_count?: number;
};
export type GpuPod = {
    resource_name?: string;
    allocated_count?: number;
    namespaced_name?: string;
};
export type ProvisionableResource = {
    instance_family?: string;
    instance_type?: string;
};
export type ClusterStatus = {
    gpu_nodes?: GpuNode[];
    provisionable_resources?: ProvisionableResource[];
    gpu_pods?: GpuPod[];
};
export type UpdateClusterStatusRequest = {
    cluster_status?: ClusterStatus;
};
export type UpdateClusterStatusResponse = {};
export declare class JobWorkerService {
    static UpdateClusterStatus(req: UpdateClusterStatusRequest, initReq?: fm.InitReq): Promise<UpdateClusterStatusResponse>;
}
