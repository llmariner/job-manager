package util

import v1 "github.com/llm-operator/job-manager/api/v1"

const jobPrefix = "job-"

// GetK8sJobName returns the the Kubernetes Job resource name from the job ID.
func GetK8sJobName(jobID string) string {
	return jobPrefix + jobID
}

// GetJobID returns the job ID from the Kubernetes Job resource name.
func GetJobID(k8sJobName string) string {
	return k8sJobName[len(jobPrefix):]
}

// GetJobNamespace returns the namespace of the job.
func GetJobNamespace(j *v1.Job) string {
	return j.OrganizationId
}
