#!/usr/bin/env bash
#
# Updates the OS packages of the node this pod runs on, for kubeaid-agent.
#
# kubeaid-agent starts one Job per node during a service window. The pod is privileged with
# hostPID and mounts the host's /opt/obmondo at the same path. This script stages the static
# linuxaid-cli onto the host and runs `system-update --agent-controlled` in the host's
# namespaces: packages only, no service window, no OpenVox, never a reboot. linuxaid-cli writes
# whether the node needs a reboot to RESULT_FILE, and this script copies it to the termination
# message, where kubeaid-agent reads it. The exit code is linuxaid-cli's.
#
# Usage: node-update.sh <result file under /opt/obmondo>
set -euo pipefail

RESULT_FILE="${1:?usage: node-update.sh <result file under /opt/obmondo>}"
CERTNAME="${CERTNAME:?CERTNAME (the cluster certname) is required}"
HOST_BIN_DIR=/opt/obmondo/bin
TERMINATION_LOG="${TERMINATION_LOG:-/dev/termination-log}"

case "${RESULT_FILE}" in
	/opt/obmondo/*) ;;
	*) echo "[node-update] result file must be under /opt/obmondo (shared with the host): ${RESULT_FILE}" >&2; exit 2 ;;
esac

# Run a command in the host's namespaces (needs pod hostPID + container privileged).
ns() { nsenter --target 1 --mount --uts --ipc --net --pid -- "$@"; }

install -d -m 0755 "${HOST_BIN_DIR}" "$(dirname "${RESULT_FILE}")"
install -m 0755 /usr/local/bin/linuxaid-cli "${HOST_BIN_DIR}/linuxaid-cli"
rm -f "${RESULT_FILE}"

rc=0
ns "${HOST_BIN_DIR}/linuxaid-cli" system-update \
	--agent-controlled \
	--certname "${CERTNAME}" \
	--result-file "${RESULT_FILE}" || rc=$?

if [ -f "${RESULT_FILE}" ]; then
	cp "${RESULT_FILE}" "${TERMINATION_LOG}"
fi

exit "${rc}"
