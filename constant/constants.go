package constant

const (
	// Obmondo API
	ObmondoAPIURL = "https://api.obmondo.com/api"

	// Puppet
	SleepTime          = 5
	PuppetPackageName  = "puppet-agent"
	PuppetPath         = "/sbin:/usr/sbin:/bin:/usr/bin:/opt/puppetlabs/puppet/bin"
	PuppetConfig       = "/etc/puppetlabs/puppet/puppet.conf"
	PuppetVersion      = "8.23.1"
	PuppetMajorVersion = "openvox8"
	PuppetCertEnv      = "PUPPETCERT"
	PuppetPrivKeyEnv   = "PUPPETPRIVKEY"
	ExternalFacterFile = "/etc/puppetlabs/facter/facts.d/new_installation.yaml"
	PuppetPrivKeyPath  = "/etc/puppetlabs/puppet/ssl/private_keys"

	// Lock and Disabled
	AgentDisabledLockFile           = "/opt/puppetlabs/puppet/cache/state/agent_disabled.lock"
	AgentRunningLockFile            = "/opt/puppetlabs/puppet/cache/state/agent_catalog_run.lock"
	DefaultPuppetServerCustomerID   = "enableit"
	DefaultPuppetServerDomainSuffix = ".puppet.obmondo.com"

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
	CobraFlagCertname     = "certname"
	CobraFlagPuppetServer = "puppet-server"
	CobraFlagReboot       = "reboot"
	CobraFlagVersion      = "version"
)

const (
	PuppetWaitForCertTimeOut = 600
)

var (
	PuppetSuccessExitCodes = []int{0, 2}
)
