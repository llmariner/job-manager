import * as fm from "../../fetch.pb";
import * as LlmarinerJobsServerV1Job_manager_server_worker from "./job_manager_server_worker.pb";
export type ClusterSummary = {
    gpu_capacity?: number;
    gpu_allocated?: number;
    gpu_pod_count?: number;
};
export type Cluster = {
    id?: string;
    name?: string;
    status?: LlmarinerJobsServerV1Job_manager_server_worker.ClusterStatus;
    summary?: ClusterSummary;
    last_updated_at?: string;
};
export type ListClustersRequest = {};
export type ListClustersResponse = {
    clusters?: Cluster[];
};
export declare class JobService {
    static ListClusters(req: ListClustersRequest, initReq?: fm.InitReq): Promise<ListClustersResponse>;
}
