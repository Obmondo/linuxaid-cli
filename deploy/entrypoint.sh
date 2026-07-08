#!/usr/bin/env bash
#
# Launcher for the LinuxAid OpenVox agent on a Kubernetes node.
#
# The pod is only a delivery + scheduling shell: it stages the static linuxaid-cli
# binary onto the host and runs the agent inside the host's namespaces via nsenter,
# because package/user management must happen on the host, not the container.
# Designed to run once (as a Job) and exit.
#
# The obmondo-clientcert is mounted and installed into the host puppet SSL tree
# under CERTNAME (already signed — nothing to enroll). If the openvox agent is
# missing on the host, it is installed first from the voxpupuli openvox8 apt repo.
#
# Two run modes:
#   - agent (default): puppet agent -t against the customer's puppetserver.
#   - apply (CONTROL_REPO_URL set): masterless puppet apply. The control-repo is
#     cloned at CONTROL_REPO_REF (the chart's linuxaid.tag; latest tag when unset)
#     into CODE_DIR under the /opt/obmondo hostPath — so the host sees it at the
#     same absolute path — and the Helm-values hiera data mounted at HIERA_SRC_DIR
#     is staged next to it as the global hiera layer.
set -euo pipefail

CERTNAME="${CERTNAME:?CERTNAME (obmondo-clientcert CN, e.g. <cluster>.<customerID>) is required}"
ENFORCE="${ENFORCE:-false}"
OPENVOX_ENVIRONMENT="${OPENVOX_ENVIRONMENT:-master}"
PUPPET_SERVER="${PUPPET_SERVER:-}"                          # agent mode only; run-openvox also resolves it from the Obmondo API
CLIENT_CERT_DIR="${CLIENT_CERT_DIR:-/obmondo-clientcert}"   # mounted obmondo-clientcert (tls.crt / tls.key / ca.crt)
OPENVOX_RELEASE_DEB="${OPENVOX_RELEASE_DEB:-https://apt.voxpupuli.org/openvox8-release-ubuntu26.04.deb}"

CONTROL_REPO_URL="${CONTROL_REPO_URL:-}"                    # apply-mode switch: git URL of the puppet control-repo (environment code)
CONTROL_REPO_REF="${CONTROL_REPO_REF:-}"                    # tag to checkout, from the chart's linuxaid.tag (default: latest tag)
HIERA_SRC_DIR="${HIERA_SRC_DIR:-/hiera-data}"               # mounted ConfigMap with the rendered Helm-values hiera data
GIT_CRED_DIR="${GIT_CRED_DIR:-/git-credentials}"            # optional secret mount: ssh-privatekey or token

OPENVOX_DIR=/opt/obmondo/openvox                            # must live under the /opt/obmondo hostPath mount
CODE_DIR="${CODE_DIR:-${OPENVOX_DIR}/code}"
DATA_DIR="${OPENVOX_DIR}/data"
HIERA_CONF="${OPENVOX_DIR}/hiera.yaml"

HOST_BIN_DIR=/opt/obmondo/bin
HOST_CLI="${HOST_BIN_DIR}/linuxaid-cli"
PUPPET_BIN=/opt/puppetlabs/bin/puppet
PUPPET_SSL=/etc/puppetlabs/puppet/ssl
PUPPET_CONF=/etc/puppetlabs/puppet/puppet.conf

# Run a command in the host's namespaces (needs pod hostPID + container privileged).
ns() { nsenter --target 1 --mount --uts --ipc --net --pid -- "$@"; }
log() { printf '[entrypoint] %s\n' "$*"; }

# git with credentials from GIT_CRED_DIR when present: an ssh-privatekey (copied to
# 0600 first — secret mounts are world-readable and ssh refuses such keys) or a
# Gitea token for https remotes.
git_cmd() {
	if [ -f "${GIT_CRED_DIR}/ssh-privatekey" ]; then
		if [ ! -f /tmp/git-ssh-key ]; then
			install -m 0600 "${GIT_CRED_DIR}/ssh-privatekey" /tmp/git-ssh-key
		fi
		GIT_SSH_COMMAND="ssh -i /tmp/git-ssh-key -o StrictHostKeyChecking=accept-new -o UserKnownHostsFile=/tmp/git-known-hosts" git "$@"
	elif [ -f "${GIT_CRED_DIR}/token" ]; then
		git -c http.extraHeader="Authorization: token $(cat "${GIT_CRED_DIR}/token")" "$@"
	else
		git "$@"
	fi
}

# Clone a repo at the given ref; without one, at its latest tag (version-sorted),
# falling back to the default-branch tip for untagged repos.
clone_repo() {
	local url=$1 dest=$2 ref=$3
	rm -rf "${dest}"
	mkdir -p "$(dirname "${dest}")"

	if [ -z "${ref}" ]; then
		ref="$(git_cmd ls-remote --refs --sort='-v:refname' --tags "${url}" | awk -F'refs/tags/' 'NR==1{print $2}')"
	fi
	if [ -n "${ref}" ]; then
		log "cloning ${url} at ${ref}"
		git_cmd clone --quiet --depth 1 --branch "${ref}" "${url}" "${dest}"
	else
		log "no tags on ${url} — cloning default-branch tip"
		git_cmd clone --quiet --depth 1 "${url}" "${dest}"
	fi
}

log "certname=${CERTNAME} enforce=${ENFORCE} env=${OPENVOX_ENVIRONMENT} mode=$([ -n "${CONTROL_REPO_URL}" ] && echo apply || echo agent)"

# 1) Stage the static CLI onto the host via the /opt/obmondo hostPath mount.
mkdir -p "${HOST_BIN_DIR}"
install -m 0755 /usr/local/bin/linuxaid-cli "${HOST_CLI}"

# 2) Install the mounted obmondo-clientcert into the host puppet SSL tree under CERTNAME.
#    (`< file` is resolved in the container mount ns, piped into a host-side writer.)
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

# 3) Install the openvox agent on the host if it is missing, from the voxpupuli
#    openvox8 apt repo. Runs on the host (Ubuntu) via nsenter.
if ! ns test -x "${PUPPET_BIN}"; then
	log "openvox agent not found on host — installing from ${OPENVOX_RELEASE_DEB}"
	ns bash -c '
		set -eu
		export DEBIAN_FRONTEND=noninteractive
		url="$1"
		deb="$(mktemp --suffix=.deb)"
		if command -v curl >/dev/null 2>&1; then
			curl -fsSL -o "$deb" "$url"
		else
			wget -qO "$deb" "$url"
		fi
		apt-get install -y "$deb"
		apt-get update
		apt-get install -y openvox-agent
		rm -f "$deb"
	' _ "${OPENVOX_RELEASE_DEB}"
fi

if [ -n "${CONTROL_REPO_URL}" ]; then
	# 4a) Masterless apply: materialize the puppet code + hiera data on the host.
	#     The clone runs in the container (which has git+ssh) into the hostPath,
	#     so the host-side puppet apply reads it at the same absolute path.
	if [ ! -f "${HIERA_SRC_DIR}/values.yaml" ]; then
		log "no hiera data at ${HIERA_SRC_DIR}/values.yaml — is the hiera ConfigMap mounted?"
		exit 1
	fi

	env_dir="${CODE_DIR}/environments/${OPENVOX_ENVIRONMENT}"
	clone_repo "${CONTROL_REPO_URL}" "${env_dir}" "${CONTROL_REPO_REF}"

	# Stage the Helm-values hiera data and point a global-layer hiera.yaml at it:
	# global beats the environment layer, so chart values win over repo defaults.
	mkdir -p "${DATA_DIR}"
	install -m 0644 "${HIERA_SRC_DIR}/values.yaml" "${DATA_DIR}/values.yaml"
	cat > "${HIERA_CONF}" <<-EOF
	version: 5
	defaults:
	  datadir: ${DATA_DIR}
	  data_hash: yaml_data
	hierarchy:
	  - name: "Helm chart values"
	    path: values.yaml
	EOF

	# Minimal puppet.conf on the host — no server: the catalog is compiled locally.
	ns install -d -m 0755 "$(dirname "${PUPPET_CONF}")"
	{
		echo "[main]"
		echo "certname = ${CERTNAME}"
		echo "environment = ${OPENVOX_ENVIRONMENT}"
		echo "report = true"
	} | ns tee "${PUPPET_CONF}" >/dev/null

	# 5a) Run masterless apply on the host. Report-only (--noop) unless ENFORCE=true.
	args=(run-openvox --apply
		--environmentpath "${CODE_DIR}/environments"
		--openvox-environment "${OPENVOX_ENVIRONMENT}"
		--hiera-config "${HIERA_CONF}")
	if [ "${ENFORCE}" = "true" ]; then args+=(--enforce); fi
	log "running: linuxaid-cli ${args[*]}"
	ns env CERTNAME="${CERTNAME}" OBMONDO_SKIP_CONNECTIVITY_METRICS=1 PATH="/opt/puppetlabs/bin:/usr/sbin:/usr/bin:/sbin:/bin" \
		"${HOST_CLI}" "${args[@]}"
else
	# 4b) Agent mode: minimal puppet.conf on the host, pointing at the puppetserver.
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

	# 5b) Run the agent on the host. Report-only (--noop) unless ENFORCE=true.
	args=(run-openvox)
	if [ "${ENFORCE}" = "true" ]; then args+=(--enforce); fi
	log "running: linuxaid-cli ${args[*]}"
	ns env CERTNAME="${CERTNAME}" OBMONDO_SKIP_CONNECTIVITY_METRICS=1 PATH="/opt/puppetlabs/bin:/usr/sbin:/usr/bin:/sbin:/bin" \
		"${HOST_CLI}" "${args[@]}"
fi
