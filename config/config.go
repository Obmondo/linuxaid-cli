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

func GetCertName() string {
	return viperConfig.GetString(constant.CobraFlagCertName)
}

func GetPupeptServer() string {
	return viperConfig.GetString(constant.CobraFlagPuppetServer)
}

func GetDebug() bool {
	return viperConfig.GetBool(constant.CobraFlagDebug)
}

func DoReboot() bool {
	return viperConfig.GetBool(constant.CobraFlagReboot)
}
