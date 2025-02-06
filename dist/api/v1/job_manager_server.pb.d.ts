import * as fm from "../../fetch.pb";
import * as LlmarinerJobsServerV1Job_manager_server_worker from "./job_manager_server_worker.pb";
export type ClusterSummary = {
    gpuCapacity?: number;
    gpuUsed?: number;
    jobCount?: number;
};
export type Cluster = {
    id?: string;
    name?: string;
    status?: LlmarinerJobsServerV1Job_manager_server_worker.ClusterStatus;
    summary?: ClusterSummary;
    lastUpdatedAt?: string;
};
export type ListClustersRequest = {};
export type ListClustersResponse = {
    clusters?: Cluster[];
};
export declare class JobService {
    static ListClusters(req: ListClustersRequest, initReq?: fm.InitReq): Promise<ListClustersResponse>;
}
