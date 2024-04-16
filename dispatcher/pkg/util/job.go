package util

const jobPrefix = "job-"

// GetK8sJobName returns the the Kubernetes Job resource name from the job ID.
func GetK8sJobName(jobID string) string {
	return jobPrefix + jobID
}

// GetJobID returns the job ID from the Kubernetes Job resource name.
func GetJobID(k8sJobName string) string {
	return k8sJobName[len(jobPrefix):]
}
