import * as fm from "../../fetch.pb";
export declare enum InternalJobState {
    STATE_UNSPECIFIED = "STATE_UNSPECIFIED",
    QUEUED = "QUEUED",
    RUNNING = "RUNNING",
    FAILED = "FAILED",
    SUCCEEDED = "SUCCEEDED",
    CANCELED = "CANCELED"
}
export declare enum InternalJobAction {
    ACTION_UNSPECIFIED = "ACTION_UNSPECIFIED",
    CREATING = "CREATING",
    CANCELING = "CANCELING"
}
export declare enum UpdateJobPhaseRequestPhase {
    PHASE_UNSPECIFIED = "PHASE_UNSPECIFIED",
    PREPROCESSED = "PREPROCESSED",
    JOB_CREATED = "JOB_CREATED",
    FINETUNED = "FINETUNED",
    FAILED = "FAILED",
    RECREATE = "RECREATE",
    CANCELED = "CANCELED"
}
export type IntegrationWandb = {
    project?: string;
    name?: string;
    entity?: string;
    tags?: string[];
};
export type Integration = {
    type?: string;
    wandb?: IntegrationWandb;
};
export type JobError = {
    code?: string;
    message?: string;
    param?: string;
};
export type JobHyperparameters = {
    batch_size?: number;
    learning_rate_multiplier?: number;
    n_epochs?: number;
};
export type JobResources = {
    gpu_count?: number;
};
export type Job = {
    id?: string;
    created_at?: string;
    error?: JobError;
    fine_tuned_model?: string;
    finished_at?: string;
    hyperparameters?: JobHyperparameters;
    model?: string;
    object?: string;
    organization_id?: string;
    result_files?: string[];
    status?: string;
    trained_tokens?: number;
    training_file?: string;
    validation_file?: string;
    integrations?: Integration[];
    seed?: number;
    project_id?: string;
    kubernetes_namespace?: string;
    cluster_id?: string;
    organization_title?: string;
    project_title?: string;
    cluster_name?: string;
    resources?: JobResources;
};
export type CreateJobRequestHyperparameters = {
    batch_size?: number;
    learning_rate_multiplier?: number;
    n_epochs?: number;
};
export type CreateJobRequest = {
    model?: string;
    training_file?: string;
    hyperparameters?: CreateJobRequestHyperparameters;
    suffix?: string;
    validation_file?: string;
    integrations?: Integration[];
    seed?: number;
};
export type ListJobsRequest = {
    after?: string;
    limit?: number;
};
export type ListJobsResponse = {
    object?: string;
    data?: Job[];
    has_more?: boolean;
    total_items?: number;
};
export type GetJobRequest = {
    id?: string;
};
export type CancelJobRequest = {
    id?: string;
};
export type InternalJob = {
    job?: Job;
    output_model_id?: string;
    suffix?: string;
    state?: InternalJobState;
    queued_action?: InternalJobAction;
};
export type ListQueuedInternalJobsRequest = {};
export type ListQueuedInternalJobsResponse = {
    jobs?: InternalJob[];
};
export type GetInternalJobRequest = {
    id?: string;
};
export type UpdateJobPhaseRequest = {
    id?: string;
    phase?: UpdateJobPhaseRequestPhase;
    message?: string;
    model_id?: string;
};
export type UpdateJobPhaseResponse = {};
export declare class FineTuningService {
    static CreateJob(req: CreateJobRequest, initReq?: fm.InitReq): Promise<Job>;
    static ListJobs(req: ListJobsRequest, initReq?: fm.InitReq): Promise<ListJobsResponse>;
    static GetJob(req: GetJobRequest, initReq?: fm.InitReq): Promise<Job>;
    static CancelJob(req: CancelJobRequest, initReq?: fm.InitReq): Promise<Job>;
}
export declare class FineTuningWorkerService {
    static ListQueuedInternalJobs(req: ListQueuedInternalJobsRequest, initReq?: fm.InitReq): Promise<ListQueuedInternalJobsResponse>;
    static GetInternalJob(req: GetInternalJobRequest, initReq?: fm.InitReq): Promise<InternalJob>;
    static UpdateJobPhase(req: UpdateJobPhaseRequest, initReq?: fm.InitReq): Promise<UpdateJobPhaseResponse>;
}
