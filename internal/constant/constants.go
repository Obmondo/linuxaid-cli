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
	// DefaultEnvironmentPath is where the per-node Job clones the puppet code for
	// masterless apply; it must live under the /opt/obmondo hostPath mount so the
	// host-side puppet sees it at the same absolute path.
	DefaultEnvironmentPath = "/opt/obmondo/openvox/code/environments"
	// DefaultHieraConfig is the global-layer hiera.yaml the Job generates; its
	// hierarchy points at the Helm-values data staged under /opt/obmondo.
	DefaultHieraConfig = "/opt/obmondo/openvox/hiera.yaml"
	// MasterlessENC is the control-repo's opensource ENC, which masterless apply runs for the
	// node parameters (hiera_datapath, obmondo_tags, ...) an opensource puppetserver gets from it.
	MasterlessENC = "linuxaid_enc.rb"

	// ServiceWindowTypeAutomatic is the booking type Obmondo reports for a scheduled service
	// window. Such a window pins the linuxaid tag its whole update cycle runs with; adhoc
	// windows are booked by hand and pin no tag.
	ServiceWindowTypeAutomatic = "automatic"

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
	// CobraFlagEnvironment mirrors puppet's own --environment/-E flag
	CobraFlagEnvironment          = "environment"
	CobraFlagEnvironmentShorthand = "E"
	// CobraFlagTag restricts a run to the given puppet tags, mapping to openvox's own --tags flag
	CobraFlagTag = "tag"
	// EnvVarOpenvoxEnvironment is the environment variable form of linuxaid-install's --environment
	// flag. It is deliberately not called ENVIRONMENT, which CI systems and deploy tooling
	// commonly set for their own purposes.
	EnvVarOpenvoxEnvironment     = "OPENVOX_ENVIRONMENT"
	CobraFlagNoReboot            = "no-reboot"
	CobraFlagSkipOpenvox         = "skip-openvox"
	CobraFlagEnforce             = "enforce"
	CobraFlagApply               = "apply"
	CobraFlagMasterless          = "masterless"
	CobraFlagEnvironmentPath     = "environmentpath"
	CobraFlagHieraConfig         = "hiera-config"
	CobraFlagSecurityExporterURL = "security-exporter-url"

	ObmondoEnv = "OBMONDO_ENV"
)

const (
	PuppetWaitForCertTimeOut = 600
)

var PuppetSuccessExitCodes = []int{0, 2, 6}
