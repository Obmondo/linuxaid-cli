package constants

const (
	// for system update script
	ObmondoAPIURL = "https://api.obmondo.com/api"
	PuppetPath    = "/sbin:/usr/sbin:/bin:/usr/bin:/usr/local/bin:/opt/puppetlabs/bin"

	AgentDisabledLockFile = "/opt/puppetlabs/puppet/cache/state/agent_disabled.lock"
	AgentRunningLockFile  = "/opt/puppetlabs/puppet/cache/state/agent_catalog_run.lock"

	// for system install script
	PuppetVersion = "7.26.0-1"
	MailTo        = "info@enableit.dk"
	PuppeetConf   = "/etc/puppetlabs/puppet/puppet.conf"
)
