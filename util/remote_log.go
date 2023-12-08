package util

import (
	"strings"

	"github.com/bitfield/script"
)

var (
	HOST     = GetHost()
	CUSTOMER = GetCustomer()
)

func Remotelog(commands ...string) *script.Pipe {
	var pipe *script.Pipe

	if webteeExists() != 0 {
		script.Echo("Webtee not found, cannot send installation logs to Obmondo servers")
		InstallFailed()
	} else {
		message := strings.Join(commands, " ")
		pipe = script.Exec("/opt/obmondo/bin/webtee --server \"api.obmondo.com:443\" --cert" + HOST + "." + CUSTOMER + " " + message + " &>/dev/null")
	}
	return pipe
}

func webteeExists() int {
	exists := script.Exec("/opt/obmondo/bin/webtee --help > /dev/null 2>&1").ExitStatus()
	return exists
} 
