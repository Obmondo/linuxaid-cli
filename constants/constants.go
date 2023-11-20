package constants

// for system update script
const (
	ObmondoAPIURL = "https://api.obmondo.com/api"
	PuppetPath    = "/sbin:/usr/sbin:/bin:/usr/bin:/usr/local/bin:/opt/puppetlabs/bin"

	AgentDisabledLockFile = "/opt/puppetlabs/puppet/cache/state/agent_disabled.lock"
	AgentRunningLockFile  = "/opt/puppetlabs/puppet/cache/state/agent_catalog_run.lock"
)
