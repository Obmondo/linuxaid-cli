package config

import (
	"gitea.obmondo.com/go-scripts/constants"
	"github.com/spf13/viper"
)

var viperConfig *viper.Viper

func Initialize() *viper.Viper {
	viperConfig = viper.New()

	return viperConfig
}

func GetCertName() string {
	return viperConfig.GetString(constants.CobraFlagCertName)
}

func GetPupeptServer() string {
	return viperConfig.GetString(constants.CobraFlagPuppetServer)
}

func GetDebug() bool {
	return viperConfig.GetBool(constants.CobraFlagDebug)
}

func DoReboot() bool {
	return viperConfig.GetBool(constants.CobraFlagReboot)
}
