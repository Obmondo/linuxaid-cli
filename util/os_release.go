package util

import (
	"log"
	"os"
	"strings"

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
	osVersion, _, _ := strings.Cut(os.Getenv("VERSION_ID"), ".")

	return osVersion
}

// List of Supported OS
func SupportedOS() {
	osVersion := os.Getenv("VERSION_ID")
	distribution := os.Getenv("NAME")

	majRelease := GetMajorRelease()

	switch distribution {
	case "Ubuntu", "Debian":
		switch osVersion {
		case osReleaseMajorVersion["ubuntu20"], osReleaseMajorVersion["ubuntu22"]:
			isInstalled := IsCaCertificateInstalled("dpkg-query -W ca-certificates openssl")

			if !isInstalled {
				apipe := script.Exec("apt update")
				apipe.Wait()
				pipe := script.Exec("apt install -y ca-certificates")
				pipe.Wait()
			}
		default:
			log.Println("Unknown Ubuntu distribution")
			os.Exit(1)
		}
	case "SUSE", "openSUSE", "SLES", "openSUSE Leap":
		switch majRelease {
		case osReleaseMajorVersion["suse12"], osReleaseMajorVersion["suse15"]:
			isInstalled := IsCaCertificateInstalled("rpm -q ca-certificates openssl ca-certificates-cacert ca-certificates-mozilla")

			if !isInstalled {
				pipe := script.Exec("zypper install -y ca-certificates openssl ca-certificates-cacert ca-certificates-mozilla")
				pipe.Wait()
			}
		default:
			log.Println("Unknown Suse distribution")
			os.Exit(1)
		}
	case "CentOS", "Red Hat Enterprise Linux Server", "Red Hat Enterprise Linux", "CentOS Linux":
		switch majRelease {
		case osReleaseMajorVersion["redhat7"], osReleaseMajorVersion["redhat8"]:
			isInstalled := IsCaCertificateInstalled("rpm -q ca-certificates openssl")

			if !isInstalled {
				pipe := script.Exec("yum install -y ca-certificates openssl")
				pipe.Wait()
			}
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
