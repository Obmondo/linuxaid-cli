package constants

const (
	// Obmondo API
	ObmondoAPIURL = "https://api.obmondo.com/api"

	// Puppet
	PuppetPackageName  = "puppet-agent"
	PuppetPath         = "/sbin:/usr/sbin:/bin:/usr/bin:/opt/puppetlabs/puppet/bin"
	PuppetConfig       = "/etc/puppetlabs/puppet/puppet.conf"
	PuppetVersion      = "7.33.0-1"
	PuppetMajorVersion = "puppet7"
	ExternalFacterFile = "/etc/puppetlabs/facter/facts.d/new_installation.yaml"

	// Lock and Disabled
	AgentDisabledLockFile         = "/opt/puppetlabs/puppet/cache/state/agent_disabled.lock"
	AgentRunningLockFile          = "/opt/puppetlabs/puppet/cache/state/agent_catalog_run.lock"
	DefaultPuppetServerCustomerID = "enableit"
	DefaultPuppetServerDomain     = "puppet.obmondo.com"

	// Progress Bar
	BarProgressSize    = 100
	BarSizeFive        = 5
	BarSizeTen         = 10
	BarSizeFifteen     = 15
	BarSizeTwenty      = 20
	BarSizeTwentyFive  = 25
	BarSizeFifty       = 50
	BarSizeSeventyFive = 75
	BarSizeHundred     = 100

	// Cobra Flags
	CobraFlagDebug        = "debug"
	CobraFlagCertName     = "certname"
	CobraFlagPuppetServer = "puppet-server"
)
