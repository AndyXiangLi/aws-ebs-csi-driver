#!/bin/bash

set -uo pipefail

BASE_DIR=$(dirname "$(realpath "${BASH_SOURCE[0]}")")
source "${BASE_DIR}"/util.sh

function ebs_check_migration() {
  KUBECONFIG=${1}

  loudecho "Checking migration"
  # There should have been no calls to the in-tree driver kubernetes.io/aws-ebs but many calls to ebs.csi.aws.com
  # Find the controller-manager log and read its metrics to verify
  NODE=$(kubectl get node -l kubernetes.io/role=master -o json --kubeconfig "${KUBECONFIG}" | jq -r ".items[].metadata.name")
  kubectl port-forward kube-controller-manager-"${NODE}" 10252:10252 -n kube-system --kubeconfig "${KUBECONFIG}" &

  # Ensure port forwarding succeeded
  n=0
  until [ "$n" -ge 30 ]; do
    set +e
    HEALTHZ=$(curl -s 127.0.0.1:10252/healthz)
    set -e
    if [[ ${HEALTHZ} == "ok" ]]; then
      loudecho "Port forwarding succeeded"
      break
    else
      loudecho "Port forwarding is not yet ready"
    fi
    n=$((n + 1))
    sleep 1
  done
  if [[ "$n" -eq 30 ]]; then
    loudecho "Timed out waiting for port forward"
    for PROC in $(jobs -p); do
      kill "${PROC}"
    done
    return 1
  fi

  set +e
  curl 127.0.0.1:10252/metrics -s | grep -a 'volume_operation_total_seconds_bucket{operation_name="provision",plugin_name="ebs.csi.aws.com"'
  CSI_CALLED=${PIPESTATUS[1]}
  set -e

  set +e
  curl 127.0.0.1:10252/metrics -s | grep -a 'volume_operation_total_seconds_bucket{operation_name="provision",plugin_name="kubernetes.io/aws-ebs"'
  INTREE_CALLED=${PIPESTATUS[1]}
  set -e

  for PROC in $(jobs -p); do
    kill "${PROC}"
  done

  loudecho "CSI_CALLED: ${CSI_CALLED}"
  loudecho "INTREE_CALLED: ${INTREE_CALLED}"

  # if CSI was called and In-tree was not called, return 0/true/success
  if [ "${CSI_CALLED}" == 0 ] && [ "${INTREE_CALLED}" == 1 ]; then
    echo 0
  else
    echo 1
  fi
}
