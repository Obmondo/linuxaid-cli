package config

import (
	"github.com/spf13/viper"
)

var viperConfig *viper.Viper

func Initialize() *viper.Viper {
	viperConfig = viper.New()

	return viperConfig
}

func GetCertName() string {
	return viperConfig.GetString("certname")
}

func GetPupeptServer() string {
	return viperConfig.GetString("puppet-server")
}

func GetDebug() bool {
	return viperConfig.GetBool("debug")
}
