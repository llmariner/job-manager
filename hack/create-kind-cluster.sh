#!/usr/bin/env bash

set -eo pipefail

base=$(readlink -f $(dirname $0))

TENANT_CLUSTER_NAME=$1
CONTROL_PLANE_NAME=$2
WORKER_PLANE_NAME=$3
WORKER_PLANE_NUM=${4:-1}

if [ ! -z "${TENANT_CLUSTER_NAME}" ]; then
  kind get clusters | grep -q ${TENANT_CLUSTER_NAME} ||\
    kind create cluster --name ${TENANT_CLUSTER_NAME} &&\
      echo "Cluster '${TENANT_CLUSTER_NAME}' created"
fi

if [ ! -z "${CONTROL_PLANE_NAME}" ]; then
  kind get clusters | grep -q ${CONTROL_PLANE_NAME} ||\
    kind create cluster --name ${CONTROL_PLANE_NAME} --config ${base}/kind-config.yaml &&\
      echo "Cluster '${CONTROL_PLANE_NAME}' created"
fi

if [ ! -z "${WORKER_PLANE_NAME}" ] && [ "${WORKER_PLANE_NAME}" != "${CONTROL_PLANE_NAME}" ]; then
  for i in $(seq 1 "${WORKER_PLANE_NUM}"); do
    kind get clusters | grep -q ${WORKER_PLANE_NAME}-${i} ||\
      kind create cluster --name ${WORKER_PLANE_NAME}-${i} &&\
        echo "Cluster '${WORKER_PLANE_NAME}-${i}' created"
  done
fi
