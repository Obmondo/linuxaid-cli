package constant

const (
	// Obmondo API
	// Puppet
	SleepTime          = 5
	PuppetPackageName  = "puppet-agent"
	PuppetPath         = "/sbin:/usr/sbin:/bin:/usr/bin:/opt/puppetlabs/puppet/bin"
	PuppetConfig       = "/etc/puppetlabs/puppet/puppet.conf"
	OpenvoxVersion     = "8.28.1"
	PuppetMajorVersion = "openvox8"
	PuppetCertEnv      = "PUPPETCERT"
	PuppetPrivKeyEnv   = "PUPPETPRIVKEY"
	InstallTokenEnv    = "TOKEN"
	ExternalFacterFile = "/etc/puppetlabs/facter/facts.d/new_installation.yaml"
	// Marker written by linuxaid-install when the node was set up without an
	// Obmondo token; linuxaid-cli skips Obmondo API calls when it exists.
	OpensourceModeFile = "/etc/obmondo/opensource-mode"
	PuppetPrivKeyPath  = "/etc/puppetlabs/puppet/ssl/private_keys"

	// Lock and Disabled
	AgentDisabledLockFile           = "/opt/puppetlabs/puppet/cache/state/agent_disabled.lock"
	AgentRunningLockFile            = "/opt/puppetlabs/puppet/cache/state/agent_catalog_run.lock"
	DefaultPuppetServerCustomerID   = "enableit"
	DefaultPuppetServerDomainSuffix = ".puppet.obmondo.com"
	DefaultOpenvoxEnv               = "master"

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
	CobraFlagDebug         = "debug"
	CobraFlagCertname      = "certname"
	CobraFlagOpenvoxServer = "puppet-server"
	// ViperKeyInstallEnvironment is the viper key behind linuxaid-install's --environment flag. It
	// deliberately does not match the flag name: viper runs with AutomaticEnv, so a key named
	// "environment" would be satisfied by any stray ENVIRONMENT variable on the host.
	ViperKeyInstallEnvironment = "install-environment"
	// CobraFlagEnvironment mirrors puppet's own --environment/-E flag on run-openvox
	CobraFlagEnvironment          = "environment"
	CobraFlagEnvironmentShorthand = "E"
	CobraFlagNoReboot             = "no-reboot"
	CobraFlagSkipOpenvox          = "skip-openvox"
	CobraFlagSecurityExporterURL  = "security-exporter-url"

	ObmondoEnv = "OBMONDO_ENV"
)

const (
	PuppetWaitForCertTimeOut = 600
)

var (
	PuppetSuccessExitCodes = []int{0, 2}
)
