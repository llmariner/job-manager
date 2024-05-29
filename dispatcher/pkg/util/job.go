package util

const (
	jobPrefix      = "job-"
	notebookPrefix = "nb-"
)

// GetK8sJobName returns the the Kubernetes Job resource name from the job ID.
func GetK8sJobName(jobID string) string {
	return jobPrefix + jobID
}

// GetJobID returns the job ID from the Kubernetes Job resource name.
func GetJobID(k8sJobName string) string {
	return k8sJobName[len(jobPrefix):]
}

// GetK8sNotebookName returns the Kubernetes deployment name from the notebook ID.
func GetK8sNotebookName(notebookID string) string {
	return notebookPrefix + notebookID
}

// GetNotebookID returns the notebook ID from the Kubernetes deployment name.
func GetNotebookID(k8sNotebookName string) string {
	return k8sNotebookName[len(notebookPrefix):]
}
