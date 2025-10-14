package config

import (
	"gitea.obmondo.com/EnableIT/go-scripts/constant"
	"github.com/spf13/viper"
)

var viperConfig *viper.Viper

func Initialize() *viper.Viper {
	viperConfig = viper.New()

	return viperConfig
}

func initIfNil() {
	if viperConfig == nil {
		Initialize()
	}
}

func GetCertName() string {
	initIfNil()
	return viperConfig.GetString(constant.CobraFlagCertName)
}

func GetPupeptServer() string {
	initIfNil()
	return viperConfig.GetString(constant.CobraFlagPuppetServer)
}

func GetDebug() bool {
	initIfNil()
	return viperConfig.GetBool(constant.CobraFlagDebug)
}

func DoReboot() bool {
	initIfNil()
	return viperConfig.GetBool(constant.CobraFlagReboot)
}
