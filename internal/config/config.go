package config

import (
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/constant"
	"github.com/spf13/viper"
)

// Config is the resolved configuration for one run. cmd/ builds it once from the flags and
// environment and hands it down, so nothing below the composition root reads global state to
// find out what it was asked to do.
type Config struct {
	Debug               bool
	Certname            string
	OpenvoxServer       string
	NoReboot            bool
	SkipOpenvox         bool
	SecurityExporterURL string
	// Opensource is true on nodes set up without an Obmondo token, which the API rejects.
	Opensource bool
}

// Load reads the flags and environment bound to viper into a Config. Certname is left empty:
// resolving it needs the machine's certificates, which is the certs package's job.
func Load() Config {
	return Config{
		Debug:               IsDebug(),
		Certname:            GetCertname(),
		OpenvoxServer:       GetOpenvoxServer(),
		NoReboot:            NoReboot(),
		SkipOpenvox:         ShouldSkipOpenvox(),
		SecurityExporterURL: GetSecurityExporterURL(),
		Opensource:          IsOpensourceMode(),
	}
}

var viperConfig *viper.Viper

func initIfNil() {
	if viperConfig == nil {
		viperConfig = viper.New()
		viperConfig.AutomaticEnv()
	}
}

func GetCertname() string {
	initIfNil()
	return viperConfig.GetString(constant.CobraFlagCertname)
}

func GetOpenvoxServer() string {
	initIfNil()
	return viperConfig.GetString(constant.CobraFlagOpenvoxServer)
}

func IsDebug() bool {
	initIfNil()
	return viperConfig.GetBool(constant.CobraFlagDebug)
}

func NoReboot() bool {
	initIfNil()
	return viperConfig.GetBool(constant.CobraFlagNoReboot)
}

func ShouldSkipOpenvox() bool {
	initIfNil()
	return viperConfig.GetBool(constant.CobraFlagSkipOpenvox)
}

func GetSecurityExporterURL() string {
	initIfNil()
	return viperConfig.GetString(constant.CobraFlagSecurityExporterURL)
}

func GetViperInstance() *viper.Viper {
	initIfNil()
	return viperConfig
}
