package constants

const (
	// for system update script
	ObmondoAPIURL = "https://api.obmondo.com/api"

	// puppet
	PuppetPath         = "/sbin:/usr/sbin:/bin:/usr/bin:/opt/puppetlabs/puppet/bin"
	PuppetConfig       = "/etc/puppetlabs/puppet/puppet.conf"
	PuppetVersion      = "7.26.0-1"
	ExternalFacterFile = "/etc/puppetlabs/facter/facts.d/new_installation.yaml"

	// Lock and Disabled
	AgentDisabledLockFile = "/opt/puppetlabs/puppet/cache/state/agent_disabled.lock"
	AgentRunningLockFile  = "/opt/puppetlabs/puppet/cache/state/agent_catalog_run.lock"

	// Progress Bar
	BarProgressSize   = 100
	BarSizeFive       = 5
	BarSizeTen        = 10
	BarSizeFifteen    = 15
	BarSizeTwenty     = 20
	BarSizeTwentyFive = 25
	BarSizeHundred    = 100
)
