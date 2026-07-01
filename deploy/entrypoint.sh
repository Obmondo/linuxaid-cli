#!/usr/bin/env bash
#
# Launcher for the LinuxAid OpenVox agent on a Kubernetes node.
#
# The pod is only a delivery + scheduling shell: it stages the static
# linuxaid-cli binary onto the host and runs the agent inside the host's
# namespaces via nsenter, because package/user/sudo management must happen on
# the host, not in the container. Designed to run once (as a Job) and exit.
#
# Cert model: the pod mounts the cluster's obmondo-clientcert; this script
# installs it into the host puppet SSL tree under CERTNAME so that both
# `puppet agent` and linuxaid-cli's Obmondo API client pick it up.
set -euo pipefail

CERTNAME="${CERTNAME:?CERTNAME (obmondo-clientcert CN, e.g. <cluster>.<customerID>) is required}"
ENFORCE="${ENFORCE:-false}"
OPENVOX_ENVIRONMENT="${OPENVOX_ENVIRONMENT:-master}"
PUPPET_SERVER="${PUPPET_SERVER:-}"                          # optional; run-openvox also resolves it from the Obmondo API
CLIENT_CERT_DIR="${CLIENT_CERT_DIR:-/obmondo-clientcert}"   # mounted obmondo-clientcert (tls.crt / tls.key / ca.crt)

HOST_BIN_DIR=/opt/obmondo/bin
PUPPET_SSL=/etc/puppetlabs/puppet/ssl
PUPPET_CONF=/etc/puppetlabs/puppet/puppet.conf

# Run a command in the host's namespaces (needs pod hostPID + container privileged).
ns() { nsenter --target 1 --mount --uts --ipc --net --pid -- "$@"; }
log() { printf '[entrypoint] %s\n' "$*"; }

log "certname=${CERTNAME} enforce=${ENFORCE} env=${OPENVOX_ENVIRONMENT} server=${PUPPET_SERVER:-<from-obmondo-api>}"

# 1) Stage the static CLI onto the host via the /opt/obmondo hostPath mount
#    (visible in the host mount namespace after nsenter).
mkdir -p "${HOST_BIN_DIR}"
install -m 0755 /usr/local/bin/linuxaid-cli "${HOST_BIN_DIR}/linuxaid-cli"

# The openvox/puppet agent must already be installed on the host — this image
# ships neither the agent nor linuxaid-install (the cert is provided pre-signed,
# so there is nothing to enroll). Fail fast with a clear message if it is absent.
if ! ns test -x /opt/puppetlabs/bin/puppet; then
	log "ERROR: /opt/puppetlabs/bin/puppet not found on the host — pre-install the openvox agent on this node."
	exit 1
fi

# 2) Install the mounted obmondo-clientcert into the host puppet SSL tree under
#    CERTNAME. The `< file` redirection is resolved in the container mount ns, so
#    the cert content is piped into a host-side writer.
ns install -d -m 0755 "${PUPPET_SSL}/certs"
ns install -d -m 0700 "${PUPPET_SSL}/private_keys"
ns tee "${PUPPET_SSL}/certs/${CERTNAME}.pem"        >/dev/null < "${CLIENT_CERT_DIR}/tls.crt"
ns tee "${PUPPET_SSL}/private_keys/${CERTNAME}.pem" >/dev/null < "${CLIENT_CERT_DIR}/tls.key"
ns chmod 0644 "${PUPPET_SSL}/certs/${CERTNAME}.pem"
ns chmod 0600 "${PUPPET_SSL}/private_keys/${CERTNAME}.pem"
if [ -f "${CLIENT_CERT_DIR}/ca.crt" ]; then
	ns tee "${PUPPET_SSL}/certs/ca.pem" >/dev/null < "${CLIENT_CERT_DIR}/ca.crt"
	ns chmod 0644 "${PUPPET_SSL}/certs/ca.pem"
fi

# 3) Minimal puppet.conf on the host.
ns install -d -m 0755 "$(dirname "${PUPPET_CONF}")"
{
	echo "[main]"
	if [ -n "${PUPPET_SERVER}" ]; then echo "server = ${PUPPET_SERVER}"; fi
	echo "certname = ${CERTNAME}"
	echo "masterport = 443"
	echo "environment = ${OPENVOX_ENVIRONMENT}"
	echo
	echo "[agent]"
	echo "report = true"
} | ns tee "${PUPPET_CONF}" >/dev/null

# 4) Run the agent on the host. Report-only (--noop) unless ENFORCE=true.
args=(run-openvox)
if [ "${ENFORCE}" = "true" ]; then args+=(--enforce); fi
log "running: linuxaid-cli ${args[*]}"
ns env CERTNAME="${CERTNAME}" PATH="/opt/puppetlabs/bin:/usr/sbin:/usr/bin:/sbin:/bin" \
	"${HOST_BIN_DIR}/linuxaid-cli" "${args[@]}"
