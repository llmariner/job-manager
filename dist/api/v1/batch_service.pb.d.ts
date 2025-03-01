import * as fm from "../../fetch.pb";
type Absent<T, K extends keyof T> = {
    [k in Exclude<keyof T, K>]?: undefined;
};
type OneOf<T> = {
    [k in keyof T]?: undefined;
} | (keyof T extends infer K ? (K extends string & keyof T ? {
    [k in K]: T[K];
} & Absent<T, K> : never) : never);
export declare enum InternalBatchJobState {
    STATE_UNSPECIFIED = "STATE_UNSPECIFIED",
    QUEUED = "QUEUED",
    RUNNING = "RUNNING",
    SUCCEEDED = "SUCCEEDED",
    FAILED = "FAILED",
    CANCELED = "CANCELED",
    DELETED = "DELETED"
}
export declare enum InternalBatchJobAction {
    ACTION_UNSPECIFIED = "ACTION_UNSPECIFIED",
    CREATING = "CREATING",
    CANCELING = "CANCELING",
    DELETING = "DELETING"
}
export type BatchJobError = {
    code?: string;
    message?: string;
};
export type BatchJobResources = {
    gpu_count?: number;
};
type BaseBatchJobKind = {};
export type BatchJobKind = BaseBatchJobKind & OneOf<{
    pytorch: PyTorchJob;
}>;
export type BatchJob = {
    id?: string;
    created_at?: string;
    finished_at?: string;
    error?: BatchJobError;
    status?: string;
    image?: string;
    command?: string;
    resources?: BatchJobResources;
    envs?: {
        [key: string]: string;
    };
    data_files?: string[];
    project_id?: string;
    kubernetes_namespace?: string;
    cluster_id?: string;
    kind?: BatchJobKind;
    organization_title?: string;
    project_title?: string;
    cluster_name?: string;
};
export type PyTorchJob = {
    worker_count?: number;
};
export type CreateBatchJobRequest = {
    image?: string;
    command?: string;
    scripts?: {
        [key: string]: Uint8Array;
    };
    resources?: BatchJobResources;
    envs?: {
        [key: string]: string;
    };
    data_files?: string[];
    kind?: BatchJobKind;
};
export type ListBatchJobsRequest = {
    after?: string;
    limit?: number;
};
export type ListBatchJobsResponse = {
    jobs?: BatchJob[];
    has_more?: boolean;
};
export type GetBatchJobRequest = {
    id?: string;
};
export type CancelBatchJobRequest = {
    id?: string;
};
export type DeleteBatchJobRequest = {
    id?: string;
};
export type InternalBatchJob = {
    job?: BatchJob;
    state?: InternalBatchJobState;
    queued_action?: InternalBatchJobAction;
};
export type ListQueuedInternalBatchJobsRequest = {};
export type ListQueuedInternalBatchJobsResponse = {
    jobs?: InternalBatchJob[];
};
export type GetInternalBatchJobRequest = {
    id?: string;
};
export type UpdateBatchJobStateRequest = {
    id?: string;
    state?: InternalBatchJobState;
    reason?: string;
    message?: string;
};
export type UpdateBatchJobStateResponse = {};
export declare class BatchService {
    static CreateBatchJob(req: CreateBatchJobRequest, initReq?: fm.InitReq): Promise<BatchJob>;
    static ListBatchJobs(req: ListBatchJobsRequest, initReq?: fm.InitReq): Promise<ListBatchJobsResponse>;
    static GetBatchJob(req: GetBatchJobRequest, initReq?: fm.InitReq): Promise<BatchJob>;
    static CancelBatchJob(req: CancelBatchJobRequest, initReq?: fm.InitReq): Promise<BatchJob>;
    static DeleteBatchJob(req: DeleteBatchJobRequest, initReq?: fm.InitReq): Promise<BatchJob>;
}
export declare class BatchWorkerService {
    static ListQueuedInternalBatchJobs(req: ListQueuedInternalBatchJobsRequest, initReq?: fm.InitReq): Promise<ListQueuedInternalBatchJobsResponse>;
    static GetInternalBatchJob(req: GetInternalBatchJobRequest, initReq?: fm.InitReq): Promise<InternalBatchJob>;
    static UpdateBatchJobState(req: UpdateBatchJobStateRequest, initReq?: fm.InitReq): Promise<UpdateBatchJobStateResponse>;
}
export {};
