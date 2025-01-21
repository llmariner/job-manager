package server

import (
	"net/http"

	"github.com/go-logr/logr"
	v40 "github.com/llmariner/job-manager/experimental/slurm/api/v0040"
)

// New returns S.
func New(
	proxy *Proxy,
	logger logr.Logger,
) *S {
	return &S{
		proxy:  proxy,
		logger: logger,
	}
}

// S is a server.
type S struct {
	proxy  *Proxy
	logger logr.Logger
}

// SlurmV0040GetDiag implements the endpoint
func (s *S) SlurmV0040GetDiag(w http.ResponseWriter, r *http.Request) {
}

// SlurmV0040PostJobSubmit implements the endpoint
func (s *S) SlurmV0040PostJobSubmit(w http.ResponseWriter, r *http.Request) {
}

// SlurmV0040DeleteJob implements the endpoint
func (s *S) SlurmV0040DeleteJob(w http.ResponseWriter, r *http.Request, jobID string, params v40.SlurmV0040DeleteJobParams) {
}

// SlurmV0040GetJob implements the endpoint
func (s *S) SlurmV0040GetJob(w http.ResponseWriter, r *http.Request, jobID string, params v40.SlurmV0040GetJobParams) {
}

// SlurmV0040PostJob implements the endpoint
func (s *S) SlurmV0040PostJob(w http.ResponseWriter, r *http.Request, jobID string) {
}

// SlurmV0040DeleteJobs implements the endpoint
func (s *S) SlurmV0040DeleteJobs(w http.ResponseWriter, r *http.Request) {
}

// SlurmV0040GetJobs implements the endpoint
func (s *S) SlurmV0040GetJobs(w http.ResponseWriter, r *http.Request, params v40.SlurmV0040GetJobsParams) {
	s.proxy.forward(w, r, http.MethodGet, "/slurm/v0.0.41/jobs/")
}

// SlurmV0040GetJobsState implements the endpoint
func (s *S) SlurmV0040GetJobsState(w http.ResponseWriter, r *http.Request, params v40.SlurmV0040GetJobsStateParams) {
}

// SlurmV0040GetLicenses implements the endpoint
func (s *S) SlurmV0040GetLicenses(w http.ResponseWriter, r *http.Request) {
}

// SlurmV0040DeleteNode implements the endpoint
func (s *S) SlurmV0040DeleteNode(w http.ResponseWriter, r *http.Request, nodeName string) {
}

// SlurmV0040GetNode implements the endpoint
func (s *S) SlurmV0040GetNode(w http.ResponseWriter, r *http.Request, nodeName string, params v40.SlurmV0040GetNodeParams) {
}

// SlurmV0040PostNode implements the endpoint
func (s *S) SlurmV0040PostNode(w http.ResponseWriter, r *http.Request, nodeName string) {
}

// SlurmV0040GetNodes implements the endpoint
func (s *S) SlurmV0040GetNodes(w http.ResponseWriter, r *http.Request, params v40.SlurmV0040GetNodesParams) {
}

// SlurmV0040GetPartition implements the endpoint
func (s *S) SlurmV0040GetPartition(w http.ResponseWriter, r *http.Request, partitionName string, params v40.SlurmV0040GetPartitionParams) {
}

// SlurmV0040GetPartitions implements the endpoint
func (s *S) SlurmV0040GetPartitions(w http.ResponseWriter, r *http.Request, params v40.SlurmV0040GetPartitionsParams) {
}

// SlurmV0040GetPing implements the endpoint
func (s *S) SlurmV0040GetPing(w http.ResponseWriter, r *http.Request) {
}

// SlurmV0040GetReconfigure implements the endpoint
func (s *S) SlurmV0040GetReconfigure(w http.ResponseWriter, r *http.Request) {
}

// SlurmV0040GetReservation implements the endpoint
func (s *S) SlurmV0040GetReservation(w http.ResponseWriter, r *http.Request, reservationName string, params v40.SlurmV0040GetReservationParams) {
}

// SlurmV0040GetReservations implements the endpoint
func (s *S) SlurmV0040GetReservations(w http.ResponseWriter, r *http.Request, params v40.SlurmV0040GetReservationsParams) {
}

// SlurmV0040GetShares implements the endpoint
func (s *S) SlurmV0040GetShares(w http.ResponseWriter, r *http.Request, params v40.SlurmV0040GetSharesParams) {
}

// SlurmdbV0040DeleteAccount implements the endpoint
func (s *S) SlurmdbV0040DeleteAccount(w http.ResponseWriter, r *http.Request, accountName string) {
}

// SlurmdbV0040GetAccount implements the endpoint
func (s *S) SlurmdbV0040GetAccount(w http.ResponseWriter, r *http.Request, accountName string, params v40.SlurmdbV0040GetAccountParams) {
}

// SlurmdbV0040GetAccounts implements the endpoint
func (s *S) SlurmdbV0040GetAccounts(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0040GetAccountsParams) {
}

// SlurmdbV0040PostAccounts implements the endpoint
func (s *S) SlurmdbV0040PostAccounts(w http.ResponseWriter, r *http.Request) {
}

// SlurmdbV0040PostAccountsAssociation implements the endpoint
func (s *S) SlurmdbV0040PostAccountsAssociation(w http.ResponseWriter, r *http.Request) {
}

// SlurmdbV0040DeleteAssociation implements the endpoint
func (s *S) SlurmdbV0040DeleteAssociation(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0040DeleteAssociationParams) {
}

// SlurmdbV0040GetAssociation implements the endpoint
func (s *S) SlurmdbV0040GetAssociation(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0040GetAssociationParams) {
}

// SlurmdbV0040DeleteAssociations implements the endpoint
func (s *S) SlurmdbV0040DeleteAssociations(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0040DeleteAssociationsParams) {
}

// SlurmdbV0040GetAssociations implements the endpoint
func (s *S) SlurmdbV0040GetAssociations(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0040GetAssociationsParams) {
}

// SlurmdbV0040PostAssociations implements the endpoint
func (s *S) SlurmdbV0040PostAssociations(w http.ResponseWriter, r *http.Request) {
}

// SlurmdbV0040DeleteCluster implements the endpoint
func (s *S) SlurmdbV0040DeleteCluster(w http.ResponseWriter, r *http.Request, clusterName string, params v40.SlurmdbV0040DeleteClusterParams) {
}

// SlurmdbV0040GetCluster implements the endpoint
func (s *S) SlurmdbV0040GetCluster(w http.ResponseWriter, r *http.Request, clusterName string, params v40.SlurmdbV0040GetClusterParams) {
}

// SlurmdbV0040GetClusters implements the endpoint
func (s *S) SlurmdbV0040GetClusters(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0040GetClustersParams) {
}

// SlurmdbV0040PostClusters implements the endpoint
func (s *S) SlurmdbV0040PostClusters(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0040PostClustersParams) {
}

// SlurmdbV0040GetConfig implements the endpoint
func (s *S) SlurmdbV0040GetConfig(w http.ResponseWriter, r *http.Request) {
}

// SlurmdbV0040PostConfig implements the endpoint
func (s *S) SlurmdbV0040PostConfig(w http.ResponseWriter, r *http.Request) {
}

// SlurmdbV0040GetDiag implements the endpoint
func (s *S) SlurmdbV0040GetDiag(w http.ResponseWriter, r *http.Request) {
}

// SlurmdbV0040GetInstance implements the endpoint
func (s *S) SlurmdbV0040GetInstance(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0040GetInstanceParams) {
}

// SlurmdbV0040GetInstances implements the endpoint
func (s *S) SlurmdbV0040GetInstances(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0040GetInstancesParams) {
}

// SlurmdbV0040GetJob implements the endpoint
func (s *S) SlurmdbV0040GetJob(w http.ResponseWriter, r *http.Request, jobID string) {
}

// SlurmdbV0040GetJobs implements the endpoint
func (s *S) SlurmdbV0040GetJobs(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0040GetJobsParams) {
}

// SlurmdbV0040GetQos implements the endpoint
func (s *S) SlurmdbV0040GetQos(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0040GetQosParams) {
}

// SlurmdbV0040PostQos implements the endpoint
func (s *S) SlurmdbV0040PostQos(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0040PostQosParams) {
}

// SlurmdbV0040DeleteSingleQos implements the endpoint
func (s *S) SlurmdbV0040DeleteSingleQos(w http.ResponseWriter, r *http.Request, qos string) {
}

// SlurmdbV0040GetSingleQos implements the endpoint
func (s *S) SlurmdbV0040GetSingleQos(w http.ResponseWriter, r *http.Request, qos string, params v40.SlurmdbV0040GetSingleQosParams) {
}

// SlurmdbV0040GetTres implements the endpoint
func (s *S) SlurmdbV0040GetTres(w http.ResponseWriter, r *http.Request) {
}

// SlurmdbV0040PostTres implements the endpoint
func (s *S) SlurmdbV0040PostTres(w http.ResponseWriter, r *http.Request) {
}

// SlurmdbV0040DeleteUser implements the endpoint
func (s *S) SlurmdbV0040DeleteUser(w http.ResponseWriter, r *http.Request, name string) {
}

// SlurmdbV0040GetUser implements the endpoint
func (s *S) SlurmdbV0040GetUser(w http.ResponseWriter, r *http.Request, name string, params v40.SlurmdbV0040GetUserParams) {
}

// SlurmdbV0040GetUsers implements the endpoint
func (s *S) SlurmdbV0040GetUsers(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0040GetUsersParams) {
}

// SlurmdbV0040PostUsers implements the endpoint
func (s *S) SlurmdbV0040PostUsers(w http.ResponseWriter, r *http.Request) {
}

// SlurmdbV0040PostUsersAssociation implements the endpoint
func (s *S) SlurmdbV0040PostUsersAssociation(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0040PostUsersAssociationParams) {
}

// SlurmdbV0040DeleteWckey implements the endpoint
func (s *S) SlurmdbV0040DeleteWckey(w http.ResponseWriter, r *http.Request, id string) {
}

// SlurmdbV0040GetWckey implements the endpoint
func (s *S) SlurmdbV0040GetWckey(w http.ResponseWriter, r *http.Request, id string) {
}

// SlurmdbV0040GetWckeys implements the endpoint
func (s *S) SlurmdbV0040GetWckeys(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0040GetWckeysParams) {
}

// SlurmdbV0040PostWckeys implements the endpoint
func (s *S) SlurmdbV0040PostWckeys(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0040PostWckeysParams) {
}
