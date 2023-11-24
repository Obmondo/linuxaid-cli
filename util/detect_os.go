package util

import (
	"fmt"
	"go-scripts/constants"
	"io"
	"log"
	"net/http"
	"regexp"

	"github.com/bitfield/script"
)

const (
	OBMONDO_WEBTEE_VERSION = constants.OBMONDO_WEBTEE_VERSION
)

var (
	PRETTY_NAME      string
	VERSION          string
	NAME             string
	VERSION_CODENAME string
	VERSION_ID       string
)

func init() {
	osReleaseVars, err := ImportOSReleaseVariables()
	if err != nil {
		log.Fatalln("Error:", err)
		return
	}

	NAME = osReleaseVars["NAME"]
	VERSION = osReleaseVars["VERSION"]
	VERSION_CODENAME = osReleaseVars["VERSION_CODENAME"]
	VERSION_ID = osReleaseVars["VERSION_ID"]
	PRETTY_NAME = osReleaseVars["PRETTY_NAME"]
}

func DetectDebian() string {
	fmt.Println("Detected " + PRETTY_NAME + " " + VERSION)

	DISTRIBUTION := NAME
	CODENAME := VERSION_CODENAME

	if DISTRIBUTION == "" || CODENAME == "" {
		log.Fatalln("ERROR: DISTRIBUTION or CODENAME field empty")
	}

	fmt.Println("Adding Obmondo GPG key to apt")
	url := "https://repos.obmondo.com/packagesign/public/apt/pubkey.gpg"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalln("Error creating HTTP request:", err)
	}
	script.Do(req).Exec("apt-key add -")

	fmt.Println("Adding Obmondo repos apt")
	script.Echo("deb https://repos.obmondo.com/packagesign/public/apt " + CODENAME + " main\"").WriteFile("/etc/apt/sources.list.d/obmondo.list")

	fmt.Println("Running apt-get update")
	script.Exec("apt-get -qq update")

	fmt.Println("Installing webtee")
	script.Exec("apt-get -qq -o Dpkg::Use-Pty=0 install -y obmondo-webtee=" + OBMONDO_WEBTEE_VERSION)

	switch CODENAME {
	case "focal", "jammy":
		// Supported distributions, do nothing
	default:
		fmt.Println("Unsupported distribution '", CODENAME, "'. Please upgrade to a supported release, or contact Obmondo for further information.")
		InstallFailed()
	}

	return CODENAME
}

func DetectRedHat() string {
	fmt.Println("Detected " + PRETTY_NAME + VERSION)

	RELEASE := extractFirstDigit(VERSION_ID)

	fmt.Println("Installing webtee")
	if script.Exec("rpm -q obmondo-webtee").ExitStatus() == 0 {
		script.Exec("yum install -y \"https://repos.obmondo.com/packagesign/public/yum/el/" + RELEASE + "/x86_64/obmondo-webtee-" + OBMONDO_WEBTEE_VERSION + "-1.x86_64.rpm\"").WithStdout(io.Discard)
	}

	return RELEASE
}

func extractFirstDigit(versionID string) string {
	re := regexp.MustCompile(`^[0-9]`)
	match := re.FindString(versionID)
	return match
}

func DetectSUSE() {
	fmt.Println("Detected %s\n" + PRETTY_NAME)

	fmt.Println("Installing webtee")
	pipe := script.Exec("rpm -q obmondo-webtee")
	pipe.WithStdout(io.Discard)
	if pipe.ExitStatus() != 0 {
		script.Exec("zypper install -y \"https://repos.obmondo.com/packagesign/public/yum/el/7/x86_64/obmondo-webtee-" + OBMONDO_WEBTEE_VERSION + "-1.x86_64.rpm\"").WithStdout(io.Discard)
	}

	switch VERSION_ID {
	case "15.3", "15.4", "15.5", "15.6":
	default:
		Remotelog("Unsupported distribution " + PRETTY_NAME + ". Please upgrade to a supported release, or contact Obmondo for further information.")
		InstallFailed()
	}
}

func InstallFailed() {
	script.Echo("Installation failed. Exiting")
	script.Echo("We have got the logs for the failed installation")
	script.Echo("Connect with Obmondo team and share your nodename, to debug further")
	log.Fatalln()
}
