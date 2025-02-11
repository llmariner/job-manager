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
    batchSize?: number;
    learningRateMultiplier?: number;
    nEpochs?: number;
};
export type Job = {
    id?: string;
    createdAt?: string;
    error?: JobError;
    fineTunedModel?: string;
    finishedAt?: string;
    hyperparameters?: JobHyperparameters;
    model?: string;
    object?: string;
    organizationId?: string;
    resultFiles?: string[];
    status?: string;
    trainedTokens?: number;
    trainingFile?: string;
    validationFile?: string;
    integrations?: Integration[];
    seed?: number;
    projectId?: string;
    kubernetesNamespace?: string;
    clusterId?: string;
    organizationTitle?: string;
    projectTitle?: string;
    clusterName?: string;
};
export type CreateJobRequestHyperparameters = {
    batchSize?: number;
    learningRateMultiplier?: number;
    nEpochs?: number;
};
export type CreateJobRequest = {
    model?: string;
    trainingFile?: string;
    hyperparameters?: CreateJobRequestHyperparameters;
    suffix?: string;
    validationFile?: string;
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
    hasMore?: boolean;
};
export type GetJobRequest = {
    id?: string;
};
export type CancelJobRequest = {
    id?: string;
};
export type InternalJob = {
    job?: Job;
    outputModelId?: string;
    suffix?: string;
    state?: InternalJobState;
    queuedAction?: InternalJobAction;
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
    modelId?: string;
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
