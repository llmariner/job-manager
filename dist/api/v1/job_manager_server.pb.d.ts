import * as fm from "../../fetch.pb";
import * as LlmarinerJobsServerV1Job_manager_server_worker from "./job_manager_server_worker.pb";
export type Cluster = {
    id?: string;
    status?: LlmarinerJobsServerV1Job_manager_server_worker.ClusterStatus;
};
export type ListClustersRequest = {};
export type ListClustersResponse = {
    clusters?: Cluster[];
};
export declare class JobService {
    static ListClusters(req: ListClustersRequest, initReq?: fm.InitReq): Promise<ListClustersResponse>;
}
