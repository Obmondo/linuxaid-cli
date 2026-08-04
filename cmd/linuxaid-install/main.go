package main

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/config"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/constant"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/helper"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/helper/logger"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/pkg/prettyfmt"
)

var Version string

var (
	debugFlag         bool
	certNameFlag      string
	openvoxServerFlag string
	openvoxEnvFlag    string
)

var rootCmd = &cobra.Command{
	Use:   "linuxaid-install",
	Short: "Setup your server with linuxaid-install",
	Example: `
	# Obmondo customers (token verified against Obmondo)
	$ TOKEN='your-token' linuxaid-install --certname web01.example --puppet-server your.openvoxserver.com

	# Opensource users (no token required)
	$ linuxaid-install --certname web01.example --puppet-server your.openvoxserver.com
	`,
	Version: Version,
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		logger.InitLogger(nil, config.IsDebug())

		// Print version first
		prettyfmt.PrettyPrintf("\n %s  %s version %s\n", prettyfmt.IconGear, prettyfmt.FontWhite(cmd.Root().Name()), prettyfmt.FontYellow(cmd.Root().Version))

		// Get certname from viper (cert, flag, or env)
		certName := helper.GetCertname()
		if certName == "" {
			errMsg := "Uh ho. I couldn't figure out the certname, please provide one as an ENV"
			prettyfmt.PrettyPrintf("\n %s %s %s\n", prettyfmt.IconCheckFail, prettyfmt.FontWhite(errMsg), prettyfmt.FontYellow("CERTNAME"))

			slog.Debug("certname is required. Provide via --certname flag or CERTNAME environment variable")
			os.Exit(1)
		}

		// Without a token, the default Obmondo shared server will never sign
		// the agent's certificate, so opensource users must point at a puppet
		// server they control.
		if _, isSet := os.LookupEnv(constant.InstallTokenEnv); !isSet {
			defaultServer := constant.DefaultPuppetServerCustomerID + constant.DefaultPuppetServerDomainSuffix
			if config.GetOpenvoxServer() == defaultServer {
				errMsg := "Uh ho. Without a TOKEN, the Obmondo server can't sign your certificate. Set the TOKEN env, or pass your own server via"
				prettyfmt.PrettyPrintf("\n %s %s %s\n", prettyfmt.IconCheckFail, prettyfmt.FontWhite(errMsg), prettyfmt.FontYellow("--puppet-server"))

				slog.Debug("no token set and puppet server is the Obmondo default. Provide TOKEN or --puppet-server")
				os.Exit(1)
			}
		}

		return nil
	},
	Run: func(*cobra.Command, []string) {
		Install()
	},
}

func init() {
	defaultServer := constant.DefaultPuppetServerCustomerID + constant.DefaultPuppetServerDomainSuffix

	rootCmd.Flags().BoolVar(&debugFlag, "debug", false, "Enable debug logs")
	rootCmd.Flags().StringVar(&certNameFlag, constant.CobraFlagCertname, "", "Certificate name (defaults to the machine's FQDN)")
	rootCmd.Flags().StringVar(&openvoxServerFlag, constant.CobraFlagOpenvoxServer, defaultServer, "Puppet server hostname")
	// no default here: it is applied in installEnvironment(), after the environment variable
	rootCmd.Flags().StringVarP(&openvoxEnvFlag, constant.CobraFlagEnvironment, constant.CobraFlagEnvironmentShorthand, "", "Openvox environment to install (Linuxaid release version, default "+constant.DefaultOpenvoxEnv+")")

	// Bind flags to viper
	v := config.GetViperInstance()
	v.BindPFlag(constant.CobraFlagDebug, rootCmd.Flags().Lookup(constant.CobraFlagDebug))
	v.BindPFlag(constant.CobraFlagCertname, rootCmd.Flags().Lookup(constant.CobraFlagCertname))
	v.BindPFlag(constant.CobraFlagOpenvoxServer, rootCmd.Flags().Lookup(constant.CobraFlagOpenvoxServer))

	// Bind environment variables
	v.BindEnv(constant.CobraFlagDebug)
	v.BindEnv(constant.CobraFlagCertname)
	v.BindEnv(constant.CobraFlagOpenvoxServer, "PUPPET_SERVER")

	// Set default values
	v.SetDefault(constant.CobraFlagOpenvoxServer, defaultServer)
}

// installEnvironment is the openvox environment the node is bootstrapped with: the --environment
// flag, then the ENVIRONMENT variable, then the default release. It is read here rather than
// through viper so the environment variable is an explicit choice instead of whatever
// viper.AutomaticEnv happens to match.
func installEnvironment() string {
	if openvoxEnvFlag != "" {
		return openvoxEnvFlag
	}

	if environment := os.Getenv(constant.EnvVarEnvironment); environment != "" {
		return environment
	}

	return constant.DefaultOpenvoxEnv
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
