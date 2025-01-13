package util

import (
	"log/slog"
	"os"
	"os/user"
	"strings"
)

// Check if the current user is root or not
// fail if user is not root
func CheckUser() {
	user, err := user.Current()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	if user.Username == "root" {
		return
	}
	slog.Error("exiting, script needs to be run as root", slog.String("current_user", user.Username))
	os.Exit(1)
}

const (
	certnameEnv   = "CERTNAME"
	puppetCertEnv = "PUPPETCERT"
	envFilePath   = "/etc/default/runPuppet"
)

func getCustomerIDFromPuppetCertString(puppetCertString string) string {
	if strings.Contains(puppetCertString, ".") && len(strings.Split(puppetCertString, ".")) == 3 && len(strings.Split(puppetCertString, ".")[1]) > 0 {
		return strings.Split(puppetCertString, ".")[1]
	}
	return ""
}

func GetCustomerIDFromEnv() string {
	certname, certnameCertExists := os.LookupEnv(certnameEnv)

	if certnameCertExists && len(certname) > 0 {
		return GetCustomerID(certname)
	}

	puppetCert, puppetCertExists := os.LookupEnv(puppetCertEnv)

	if puppetCertExists && len(puppetCert) > 0 {
		return getCustomerIDFromPuppetCertString(puppetCert)
	}

	return ""
}
