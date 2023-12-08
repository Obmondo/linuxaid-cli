package util

import (
	"fmt"
	"log"
	"os"

	"github.com/bitfield/script"
)

// List of Supported OSs
const (
	redhat7  = "7"
	redhat8  = "8"
	suse15   = "15"
	suse12   = "12"
	ubuntu22 = "22.04"
	ubuntu20 = "20.04"
)

var osReleaseMajorVersion = map[string]string{
	"redhat7":  redhat7,
	"redhat8":  redhat8,
	"suse15":   suse15,
	"suse12":   suse12,
	"ubuntu22": ubuntu22,
	"ubuntu20": ubuntu20,
}

func GetMajorRelease() string {
	osVersion := os.Getenv("VERSION")

	cmd := fmt.Sprintf("echo %s | cut -d '.' -f1'", osVersion)
	release, _ := script.Exec(cmd).String()

	return release
}

// List of Supported OS
func SupportedOS() {
	distribution := os.Getenv("NAME")
	osVersion := os.Getenv("VERSION")
	majRelease := GetMajorRelease()

	switch distribution {
	case "Ubuntu", "Debian":
		switch osVersion {
		case osReleaseMajorVersion["ubuntu20"], osReleaseMajorVersion["ubuntu22"]:
			//
		default:
			log.Println("Unknown Ubuntu distribution")
			os.Exit(1)
		}
	case "SUSE", "openSUSE", "SLES", "openSUSE Leap":
		switch majRelease {
		case osReleaseMajorVersion["suse12"], osReleaseMajorVersion["suse15"]:
			//
		default:
			log.Println("Unknown Suse distribution")
			os.Exit(1)
		}
	case "CentOS", "Red Hat Enterprise Linux Server", "Red Hat Enterprise Linux":
		switch majRelease {
		case osReleaseMajorVersion["redhat7"], osReleaseMajorVersion["redhat8"]:
			//
		default:
			log.Println("Unknown RedHat distribution")
			os.Exit(1)
		}
		//
	default:
		log.Println("Unknown distribution")
		os.Exit(1)
	}
}
