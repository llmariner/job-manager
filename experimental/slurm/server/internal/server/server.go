package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-logr/logr"
	v40 "github.com/llmariner/job-manager/experimental/slurm/api/v0041"
)

// New returns S.
func New(
	proxies []*Proxy,
	logger logr.Logger,
) *S {
	return &S{
		proxies: proxies,
		logger:  logger,
	}
}

// S is a server.
type S struct {
	proxies []*Proxy
	logger  logr.Logger

	jobSubmissionIndex int
}

// nextProxy returns the next proxy to use with simple round-robin.
func (s *S) nextProxy() *Proxy {
	p := s.proxies[s.jobSubmissionIndex]
	s.jobSubmissionIndex = (s.jobSubmissionIndex + 1) % len(s.proxies)
	return p
}

func (s *S) forwardResponse(w http.ResponseWriter, resp *http.Response) {
	w.WriteHeader(resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := io.Copy(w, bytes.NewBuffer(body)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// SlurmV0041GetDiag implements the endpoint.
func (s *S) SlurmV0041GetDiag(w http.ResponseWriter, r *http.Request) {
}

// SlurmV0041PostJobAllocate implements the endpoint.
func (s *S) SlurmV0041PostJobAllocate(w http.ResponseWriter, r *http.Request) {
}

// SlurmV0041PostJobSubmit implements the endpoint.
func (s *S) SlurmV0041PostJobSubmit(w http.ResponseWriter, r *http.Request) {
	p := s.nextProxy()
	resp, err := p.forward(w, r, http.MethodPost, "/slurm/v0.0.41/job/submit")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	s.forwardResponse(w, resp)
}

// SlurmV0041DeleteJob implements the endpoint.
func (s *S) SlurmV0041DeleteJob(w http.ResponseWriter, r *http.Request, jobID string, params v40.SlurmV0041DeleteJobParams) {
}

// SlurmV0041GetJob implements the endpoint.
func (s *S) SlurmV0041GetJob(w http.ResponseWriter, r *http.Request, jobID string, params v40.SlurmV0041GetJobParams) {
}

// SlurmV0041PostJob implements the endpoint.
func (s *S) SlurmV0041PostJob(w http.ResponseWriter, r *http.Request, jobID string) {
}

// SlurmV0041DeleteJobs implements the endpoint.
func (s *S) SlurmV0041DeleteJobs(w http.ResponseWriter, r *http.Request) {
}

// SlurmV0041GetJobs implements the endpoint.
func (s *S) SlurmV0041GetJobs(w http.ResponseWriter, r *http.Request, params v40.SlurmV0041GetJobsParams) {
	// Obtain the result from all proxies and aggregate them.

	var (
		aggResp v40.V0041OpenapiJobInfoResp
		isFirst = true
	)
	for _, p := range s.proxies {
		resp, err := p.forward(w, r, http.MethodGet, "/slurm/v0.0.41/jobs/")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.logger.Info("Received response", "proxy", p.name, "status", resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if resp.StatusCode != http.StatusOK {
			w.WriteHeader(resp.StatusCode)
			if _, err := io.Copy(w, bytes.NewBuffer(body)); err != nil {
				s.logger.Error(err, "Failed to write response body")
			}
			return
		}

		// Parse and aggregate the response.
		var jr v40.V0041OpenapiJobInfoResp
		if err := json.Unmarshal(body, &jr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if isFirst {
			s.logger.V(1).Info("First response", "proxy", p.name, "resp", jr)
			aggResp = jr
			isFirst = false
		} else {
			// Just append to jobs.
			s.logger.V(1).Info("Subsequent response", "proxy", p.name, "resp", jr)
			aggResp.Jobs = append(aggResp.Jobs, jr.Jobs...)
		}
	}

	w.WriteHeader(http.StatusOK)

	// marshal the response again.
	b, err := json.Marshal(aggResp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(w, bytes.NewBuffer(b)); err != nil {
		s.logger.Error(err, "Failed to write response body")
	}
	return
}

// SlurmV0041GetJobsState implements the endpoint.
func (s *S) SlurmV0041GetJobsState(w http.ResponseWriter, r *http.Request, params v40.SlurmV0041GetJobsStateParams) {
}

// SlurmV0041GetLicenses implements the endpoint.
func (s *S) SlurmV0041GetLicenses(w http.ResponseWriter, r *http.Request) {
}

// SlurmV0041DeleteNode implements the endpoint.
func (s *S) SlurmV0041DeleteNode(w http.ResponseWriter, r *http.Request, nodeName string) {
}

// SlurmV0041GetNode implements the endpoint.
func (s *S) SlurmV0041GetNode(w http.ResponseWriter, r *http.Request, nodeName string, params v40.SlurmV0041GetNodeParams) {
}

// SlurmV0041PostNode implements the endpoint.
func (s *S) SlurmV0041PostNode(w http.ResponseWriter, r *http.Request, nodeName string) {
}

// SlurmV0041GetNodes implements the endpoint.
func (s *S) SlurmV0041GetNodes(w http.ResponseWriter, r *http.Request, params v40.SlurmV0041GetNodesParams) {
}

// SlurmV0041GetPartition implements the endpoint.
func (s *S) SlurmV0041GetPartition(w http.ResponseWriter, r *http.Request, partitionName string, params v40.SlurmV0041GetPartitionParams) {
}

// SlurmV0041GetPartitions implements the endpoint.
func (s *S) SlurmV0041GetPartitions(w http.ResponseWriter, r *http.Request, params v40.SlurmV0041GetPartitionsParams) {
}

// SlurmV0041GetPing implements the endpoint.
func (s *S) SlurmV0041GetPing(w http.ResponseWriter, r *http.Request) {
}

// SlurmV0041GetReconfigure implements the endpoint.
func (s *S) SlurmV0041GetReconfigure(w http.ResponseWriter, r *http.Request) {
}

// SlurmV0041GetReservation implements the endpoint.
func (s *S) SlurmV0041GetReservation(w http.ResponseWriter, r *http.Request, reservationName string, params v40.SlurmV0041GetReservationParams) {
}

// SlurmV0041GetReservations implements the endpoint.
func (s *S) SlurmV0041GetReservations(w http.ResponseWriter, r *http.Request, params v40.SlurmV0041GetReservationsParams) {
}

// SlurmV0041GetShares implements the endpoint.
func (s *S) SlurmV0041GetShares(w http.ResponseWriter, r *http.Request, params v40.SlurmV0041GetSharesParams) {
}

// SlurmdbV0041DeleteAccount implements the endpoint.
func (s *S) SlurmdbV0041DeleteAccount(w http.ResponseWriter, r *http.Request, accountName string) {
}

// SlurmdbV0041GetAccount implements the endpoint.
func (s *S) SlurmdbV0041GetAccount(w http.ResponseWriter, r *http.Request, accountName string, params v40.SlurmdbV0041GetAccountParams) {
}

// SlurmdbV0041GetAccounts implements the endpoint.
func (s *S) SlurmdbV0041GetAccounts(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0041GetAccountsParams) {
}

// SlurmdbV0041PostAccounts implements the endpoint.
func (s *S) SlurmdbV0041PostAccounts(w http.ResponseWriter, r *http.Request) {
}

// SlurmdbV0041PostAccountsAssociation implements the endpoint.
func (s *S) SlurmdbV0041PostAccountsAssociation(w http.ResponseWriter, r *http.Request) {
}

// SlurmdbV0041DeleteAssociation implements the endpoint.
func (s *S) SlurmdbV0041DeleteAssociation(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0041DeleteAssociationParams) {
}

// SlurmdbV0041GetAssociation implements the endpoint.
func (s *S) SlurmdbV0041GetAssociation(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0041GetAssociationParams) {
}

// SlurmdbV0041DeleteAssociations implements the endpoint.
func (s *S) SlurmdbV0041DeleteAssociations(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0041DeleteAssociationsParams) {
}

// SlurmdbV0041GetAssociations implements the endpoint.
func (s *S) SlurmdbV0041GetAssociations(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0041GetAssociationsParams) {
}

// SlurmdbV0041PostAssociations implements the endpoint.
func (s *S) SlurmdbV0041PostAssociations(w http.ResponseWriter, r *http.Request) {
}

// SlurmdbV0041DeleteCluster implements the endpoint.
func (s *S) SlurmdbV0041DeleteCluster(w http.ResponseWriter, r *http.Request, clusterName string, params v40.SlurmdbV0041DeleteClusterParams) {
}

// SlurmdbV0041GetCluster implements the endpoint.
func (s *S) SlurmdbV0041GetCluster(w http.ResponseWriter, r *http.Request, clusterName string, params v40.SlurmdbV0041GetClusterParams) {
}

// SlurmdbV0041GetClusters implements the endpoint.
func (s *S) SlurmdbV0041GetClusters(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0041GetClustersParams) {
}

// SlurmdbV0041PostClusters implements the endpoint.
func (s *S) SlurmdbV0041PostClusters(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0041PostClustersParams) {
}

// SlurmdbV0041GetConfig implements the endpoint.
func (s *S) SlurmdbV0041GetConfig(w http.ResponseWriter, r *http.Request) {
}

// SlurmdbV0041PostConfig implements the endpoint.
func (s *S) SlurmdbV0041PostConfig(w http.ResponseWriter, r *http.Request) {
}

// SlurmdbV0041GetDiag implements the endpoint.
func (s *S) SlurmdbV0041GetDiag(w http.ResponseWriter, r *http.Request) {
}

// SlurmdbV0041GetInstance implements the endpoint.
func (s *S) SlurmdbV0041GetInstance(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0041GetInstanceParams) {
}

// SlurmdbV0041GetInstances implements the endpoint.
func (s *S) SlurmdbV0041GetInstances(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0041GetInstancesParams) {
}

// SlurmdbV0041GetJob implements the endpoint.
func (s *S) SlurmdbV0041GetJob(w http.ResponseWriter, r *http.Request, jobID string) {
}

// SlurmdbV0041GetJobs implements the endpoint.
func (s *S) SlurmdbV0041GetJobs(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0041GetJobsParams) {
}

// SlurmdbV0041GetQos implements the endpoint.
func (s *S) SlurmdbV0041GetQos(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0041GetQosParams) {
}

// SlurmdbV0041PostQos implements the endpoint.
func (s *S) SlurmdbV0041PostQos(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0041PostQosParams) {
}

// SlurmdbV0041DeleteSingleQos implements the endpoint.
func (s *S) SlurmdbV0041DeleteSingleQos(w http.ResponseWriter, r *http.Request, qos string) {
}

// SlurmdbV0041GetSingleQos implements the endpoint.
func (s *S) SlurmdbV0041GetSingleQos(w http.ResponseWriter, r *http.Request, qos string, params v40.SlurmdbV0041GetSingleQosParams) {
}

// SlurmdbV0041GetTres implements the endpoint.
func (s *S) SlurmdbV0041GetTres(w http.ResponseWriter, r *http.Request) {
}

// SlurmdbV0041PostTres implements the endpoint.
func (s *S) SlurmdbV0041PostTres(w http.ResponseWriter, r *http.Request) {
}

// SlurmdbV0041DeleteUser implements the endpoint.
func (s *S) SlurmdbV0041DeleteUser(w http.ResponseWriter, r *http.Request, name string) {
}

// SlurmdbV0041GetUser implements the endpoint.
func (s *S) SlurmdbV0041GetUser(w http.ResponseWriter, r *http.Request, name string, params v40.SlurmdbV0041GetUserParams) {
}

// SlurmdbV0041GetUsers implements the endpoint.
func (s *S) SlurmdbV0041GetUsers(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0041GetUsersParams) {
}

// SlurmdbV0041PostUsers implements the endpoint.
func (s *S) SlurmdbV0041PostUsers(w http.ResponseWriter, r *http.Request) {
}

// SlurmdbV0041PostUsersAssociation implements the endpoint.
func (s *S) SlurmdbV0041PostUsersAssociation(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0041PostUsersAssociationParams) {
}

// SlurmdbV0041DeleteWckey implements the endpoint.
func (s *S) SlurmdbV0041DeleteWckey(w http.ResponseWriter, r *http.Request, id string) {
}

// SlurmdbV0041GetWckey implements the endpoint.
func (s *S) SlurmdbV0041GetWckey(w http.ResponseWriter, r *http.Request, id string) {
}

// SlurmdbV0041GetWckeys implements the endpoint.
func (s *S) SlurmdbV0041GetWckeys(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0041GetWckeysParams) {
}

// SlurmdbV0041PostWckeys implements the endpoint.
func (s *S) SlurmdbV0041PostWckeys(w http.ResponseWriter, r *http.Request, params v40.SlurmdbV0041PostWckeysParams) {
}
