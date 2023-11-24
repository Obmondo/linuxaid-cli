package constants

const (
	// for system update script
	ObmondoAPIURL = "https://api.obmondo.com/api"
	PuppetPath    = "/sbin:/usr/sbin:/bin:/usr/bin:/usr/local/bin:/opt/puppetlabs/bin"

	AgentDisabledLockFile = "/opt/puppetlabs/puppet/cache/state/agent_disabled.lock"
	AgentRunningLockFile  = "/opt/puppetlabs/puppet/cache/state/agent_catalog_run.lock"

	// for system install script
	PATH                   = "/bin:/sbin:/usr/bin:/usr/sbin:/opt/puppetlabs/puppet/bin:/opt/obmondo/bin"
	PUPPET_VERSION         = "7.26.0-1"
	MAILTO                 = "info@enableit.dk"
	PUPPET_CONF            = "/etc/puppetlabs/puppet/puppet.conf"
	OBMONDO_WEBTEE_VERSION = "1.0.2"
	SERVER_ADDRESS         = "api.obmondo.com:443"
)
