package config

import (
	"gitea.obmondo.com/EnableIT/linuxaid-cli/constant"
	"github.com/spf13/viper"
)

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

func GetPupeptServer() string {
	initIfNil()
	return viperConfig.GetString(constant.CobraFlagPuppetServer)
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

func GetViperInstance() *viper.Viper {
	initIfNil()
	return viperConfig
}
