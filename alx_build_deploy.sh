#!/bin/bash
set -e -o pipefail

# Check if XVERSION is already exported
if [ -n "$XVERSION" ]; then
  # Extract the version prefix and increment the number
  prefix="${XVERSION%.*}" # Extract everything before the last dot
  version_number="${XVERSION##*.}" # Extract everything after the last dot
  
  # Increment the version number
  if [[ "$version_number" =~ ^[0-9]+$ ]]; then
    version_number=$((version_number + 1))
  else
    echo "Non-numeric version part detected, resetting to 0."
    version_number=0
  fi
  
  # Set the new version
  export XVERSION="${prefix}.${version_number}"
else
  # If XVERSION is not set, set the initial version
  export XVERSION="alx.1"
fi

# Output the new version
echo "XVERSION: $XVERSION"

TAG=${XVERSION} make build-docker-dispatcher build-docker-syncer
for cluster in "tenant-cluster" "gpu-worker-cluster-large" "gpu-worker-cluster-small"; do
  kind load docker-image -n${cluster}  llmariner/job-manager-dispatcher:$XVERSION
  kind load docker-image -n${cluster}  llmariner/job-manager-syncer:$XVERSION
done
for cluster in "kind-gpu-worker-cluster-large" "kind-gpu-worker-cluster-small"; do
  pod_name=$(kubectl --context=${cluster} -n llmariner get pods --no-headers -o custom-columns=":metadata.name" | grep '^job-manager-dispatcher')
  kubectl --context=${cluster}  -n llmariner patch pods/${pod_name} -p '{"spec":{"containers":[{"name":"job-manager-dispatcher","image":"llmariner/job-manager-dispatcher:'$XVERSION'"}]}}'
done
for cluster in "kind-tenant-cluster" ; do
  pod_name=$(kubectl --context=${cluster} -n llmariner get pods --no-headers -o custom-columns=":metadata.name" | grep '^job-manager-syncer')
  kubectl --context=${cluster}  -n llmariner patch pods/${pod_name} -p '{"spec":{"containers":[{"name":"job-manager-syncer","image":"llmariner/job-manager-syncer:'$XVERSION'"}]}}'
done

kubectl get pods -n llmariner -w
