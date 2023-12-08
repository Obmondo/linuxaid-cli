package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go-scripts/constants"
	"go-scripts/pkg/puppet"
	"go-scripts/util"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bitfield/script"
)

const (
	PUPPET_VERSION         = constants.PUPPET_VERSION
	MAILTO                 = constants.MAILTO
	PATH                   = constants.PATH
	PUPPET_CONF            = constants.PUPPET_CONF
	OBMONDO_WEBTEE_VERSION = constants.OBMONDO_WEBTEE_VERSION
	SERVER_ADDRESS         = constants.SERVER_ADDRESS
)

var (
	HOST               = util.GetHost()
	CUSTOMER           = util.GetCustomer()
	NAME               string
	KEY_FILE           string
	CERT_FILE          string
	SUBSCRIPTION_LEVEL string
	CERTNAME           string
	RELEASE            string
	CODENAME           string
	FAMILY             string
)

func init() {
	NAME = util.NAME
}

var (
	FRAME          = []string{"-", "\\", "|", "/"}
	FRAME_INTERVAL = 250 * time.Millisecond
)

func createTemporaryLogFile() string {
	pipe := script.Exec("mkdir -p /opt/obmondo/logs && mktemp \"/opt/obmondo/logs/obmondo-$(TZ='' date +%FT%TZ).XXXX\"")
	TMPFILE, err := pipe.String()
	if err != nil {
		script.Echo("Could not create temporary directory. Stopping before we destroy stuff.")
		util.InstallFailed()
	}
	return TMPFILE
}

type Step struct {
	Name string
	Func func()
}

func runStep(step Step, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Printf("\r[   invalid option] %s ...", step.Name)
	step.Func()
	fmt.Printf("\r[ ✔ ] %s\n", step.Name)
}

func spinnerStart(steps []Step) {
	script.Exec("tput civis --invisible")

	var wg sync.WaitGroup

	for _, step := range steps {
		wg.Add(1)
		go runStep(step, &wg)

		for k := 0; k < len(FRAME); k++ {
			fmt.Printf("\r[ %s ]", FRAME[k])
			time.Sleep(FRAME_INTERVAL)
		}
	}

	wg.Wait()

	script.Exec("tput cnorm -- normal")
}

func validateAndSetCustomerID() {
	if len(CUSTOMER) < 7 || CUSTOMER[:2] == "__" {
		if script.Exec("tty").WithStdout(io.Discard).ExitStatus() != 0 {
			reader := bufio.NewReader(os.Stdin)
			for {
				fmt.Print("Enter customer ID: ")
				CUSTOMER, _ := reader.ReadString('\n')
				CUSTOMER = CUSTOMER[:len(CUSTOMER)-1] // Remove newline character

				regexPattern := `^(enableit|[a-z]{7}|[a-z0-9]{10})$`
				re, err := regexp.Compile(regexPattern)
				if err != nil {
					log.Fatalln("Error compiling regular expression: " + err.Error() + "\n")
					return
				}

				if script.Echo(CUSTOMER).MatchRegexp(re).ExitStatus() == 0 {
					break
				} else {
					fmt.Println("Customer ID not set or too short. Please set the environment variable CUSTOMER to your customer ID and re-run the script.")
					os.Exit(2)
				}
			}
		}
	}
}

func installPrerequisitesDebian() {
	fmt.Println("Installing https transport for apt")

	util.Remotelog("apt-get -q update")

	if script.Exec("dpkg-query -q --show apt-transport-https ca-certificates").WithStdout(io.Discard).ExitStatus() == 0 {
		util.Remotelog("apt-get install iptables wget apt-transport-https ca-certificates -y")
	}

	fmt.Println("Installing gnupg")
	util.Remotelog("apt-get install gnupg -y")

	var PUPPET_DEB string
	if script.Exec("! dpkg-query --show puppet-agent | awk '{print $NF;}' | grep -q "+PUPPET_VERSION+" &>/dev/null").ExitStatus() == 0 {
		log.Fatalln("Downloading Puppet")
		PUPPET_DEB = createTemporaryLogFile() + "/puppet.deb"

		url := "https://repos.obmondo.com/puppetlabs/apt/pool/" + CODENAME + "/puppet7/p/puppet-agent/puppet-agent_" + PUPPET_VERSION + CODENAME + "_amd64.deb"
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatalln("Error creating HTTP request:", err)
			return
		}
		_, err = script.Do(req).WriteFile(PUPPET_DEB)
		if err != nil {
			log.Fatalln("Error:", err)
			return
		}
		script.Echo("Downloading Puppet Failed")
		util.InstallFailed()
	}

	fmt.Println("Installing Puppet")
	if util.Remotelog("dpkg -i ", PUPPET_DEB).ExitStatus() != 0 {
		script.Echo("Puppet installation failed.")
		util.InstallFailed()
	}
}

func subscriptionManager(reponame string) {
	fmt.Println("Enabling %s\n" + reponame)

	if script.Exec("subscription-manager repos --list-enabled").Match(reponame).ExitStatus() == 0 {
		script.Exec("subscription-manager repos --enable " + reponame).WriteFile("/dev/null")
	}
}

func installPrerequisitesRedHat() {
	release, _ := strconv.Atoi(RELEASE)
	pattern := regexp.MustCompile(`CentOS|Red Hat`)
	DISTRIBUTION, _ := script.Echo(NAME).MatchRegexp(pattern).String()
	if strings.Contains(DISTRIBUTION, "CentOS") || strings.Contains(DISTRIBUTION, "Red Hat") {
		DISTRIBUTION = strings.ReplaceAll(NAME, " ", "")
	}

	if DISTRIBUTION == "RedHat" {
		switch release {
		case 7:
			subscriptionManager("rhel-7-server-extras-rpms")
			subscriptionManager("rhel-7-server-optional-rpms")
		case 8:
			subscriptionManager("rhel-8-for-x86_64-baseos-rpms")
			subscriptionManager("rhel-8-for-x86_64-appstream-rpms")
		default:
			fmt.Println("Unsupported distribution " + DISTRIBUTION + " " + RELEASE + " . Please upgrade to a supported release or contact Obmondo for further information.")
			os.Exit(10)
		}
	} else {
		fmt.Println("Installing EPEL")
		util.Remotelog("yum", "-y", "install", "epel-release")
	}

	fmt.Println("Installing gnupg")
	util.Remotelog("yum", "-y", "install", "gnupg")

	fmt.Println("Installing Puppet and dependencies")
	if script.Exec("rpm -q puppet-agent >/dev/null").ExitStatus() != 0 {
		puppetAgentURL := "https://repos.obmondo.com/puppetlabs/yum/puppet7/el/" + RELEASE + "/x86_64/puppet-agent-" + PUPPET_VERSION + ".el" + RELEASE + ".x86_64.rpm"
		util.Remotelog("yum", "install", "-y", "iptables", puppetAgentURL)
	}
}

func installPrerequisitesSUSE() {
	fmt.Println("Adding puppetlab gpg key")
	script.Get("https://repos.obmondo.com/puppetlabs/public.key").Exec("gpg --import")

	fmt.Println("GPG key to apt")
	script.Get("https://repos.obmondo.com/packagesign/public/apt/pubkey.gpg").Exec("gpg --import")

	fmt.Println("Installing gnupg")
	util.Remotelog("zypper", "install", "-y", "gpg2")

	fmt.Println("Installing Puppet and dependencies")
	if script.Exec("rpm -q puppet-agent >/dev/null").ExitStatus() == 0 {
		puppetAgentURL := "https://repos.obmondo.com/puppetlabs/sles/puppet7/15/x86_64/puppet-agent-" + PUPPET_VERSION + ".sles15.x86_64.rpm"
		util.Remotelog("zypper", "install", "-y", "iptables", puppetAgentURL)
	}
}

func configurePuppet() {
	fmt.Println("Configuring Puppet...")

	config := `
[main]
    server = puppet.enableit.dk
		certname = ` + HOST + `.` + CUSTOMER + `
		stringify_facts = false
		masterport = 443
	
[agent]
		report = true
		pluginsync = true
		noop = true
`
	_, err := script.Echo(config).WriteFile(PUPPET_CONF)
	if err != nil {
		log.Fatalln("Can not create puppet configuration file: " + err.Error() + "\n")
	}

	fmt.Println("Setting up the external facter for a fresh installation on a new node")
	script.Exec("mkdir -p /etc/puppetlabs/facter/facts.d")

	newInstallationYAML := `
---
install_date: ` + time.Now().Format("20060102") + `
`
	_, err = script.Echo(newInstallationYAML).AppendFile(PUPPET_CONF)
	if err != nil {
		fmt.Println("Can not edit puppet configuration file: " + err.Error() + "\n")
	}

	fmt.Println("Disabling Puppet service")
	util.Remotelog("puppet", "resource", "service", "puppet", "ensure=stopped", "enable=false")

	// Check to see if Puppet is disabled because it's a new installation -- we
	// don't want to enable Puppet if it's disabled by someone else.
	if puppet.PuppetDisabled() {
		if puppet.PuppetDisabledNewInstall() {
			util.Remotelog("puppet", "agent", "--enable")
		} else {
			log.Fatalln("The Puppet agent has been disabled on the system. Please re-enable it to continue the setup.")
		}
	}
}

func installPrerequisites() {
	FAMILY = NAME
	fmt.Println("FAMILY: " + NAME)
	switch FAMILY {
	case "debian":
		installPrerequisitesDebian()
	case "redhat":
		installPrerequisitesRedHat()
	case "suse":
		installPrerequisitesSUSE()
	default:
		log.Fatalln("Unable to detect distribution; exiting.")
	}
}

func addSubscription() {
	puppet.IsPuppetInstalled()

	fmt.Println("Adding " + SUBSCRIPTION_LEVEL + " subscription for " + CERTNAME)
	reqBody := fmt.Sprintf("{\"certname\": \"%s\", \"customer_id\": \"%s\", \"product_id\": \"%s\", \"order_type\": \"subscription\"}", CERTNAME, CUSTOMER, SUBSCRIPTION_LEVEL)
	req, err := http.NewRequest("POST", "https://api.obmondo.com/api/order", bytes.NewBufferString(reqBody))
	if err != nil {
		log.Fatalln("Error creating request:", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Key", KEY_FILE)
	req.Header.Set("Cert", CERT_FILE)

	// TODO: check if subs is there or not and only then try to add it.
	script.Do(req)
}

func main() {
	validateAndSetCustomerID()

	var (
		helpFlag         = flag.Bool("h", false, "HELP")
		keyFlag          = flag.String("k", "", "KEY_FILE")
		certFlag         = flag.String("c", "", "CERT_FILE")
		subscriptionFlag = flag.String("s", "", "SUBSCRIPTION_LEVEL")
	)

	flag.Parse()

	if *helpFlag {
		script.Echo("Usage: " + os.Args[0] + " [-d] [-h] [-k KEY_FILE -c CERT_FILE -s SUBSCRIPTION_LEVEL]")
		os.Exit(0)
	}

	if *keyFlag != "" {
		KEY_FILE = *keyFlag
	}

	if *certFlag != "" {
		fmt.Println("Cert file specified:", certFlag)
	}

	if *subscriptionFlag != "" {
		switch *subscriptionFlag {
		case "unmanaged", "bronze", "silver", "gold", "platinum":
			SUBSCRIPTION_LEVEL = *subscriptionFlag
		default:
			script.Echo("ERROR: Subscription level invalid! Must be one of these; unmanaged, bronze, silver, gold, platinum")
			util.Remotelog("echo", "ERROR: Subscription level invalid! must be one of these; unmanaged, bronze, silver, gold, platinum")
			util.InstallFailed()
		}
	}

	if flag.NArg() > 0 {
		pipe := script.Echo("Invalid option - " + flag.Arg(0))
		fmt.Fprintln(os.Stderr, pipe)
		os.Exit(2)
	}

	switch NAME {
	case "Red Hat Enterprise Linux", "Red Hat Enterprise Linux Server":
		NAME = "redhat"
		RELEASE = util.DetectRedHat()
	case "Ubuntu":
		NAME = "debian"
		CODENAME = util.DetectDebian()
	case "SLES":
		NAME = "suse"
		util.DetectSUSE()
	default:
		log.Fatalln("Unable to detect distribution; exiting.")
	}

	if CERT_FILE != "" || KEY_FILE != "" {
		if (script.IfExists(CERT_FILE).ExitStatus() != 0) || (script.IfExists(KEY_FILE).ExitStatus() != 0) {
			fmt.Println("ERROR: cert or key argument is not a file")
		}
	}

	if script.IfExists(CERT_FILE).ExitStatus() == 0 {
		certName, err := script.Exec("openssl x509 -text -noout -in " + CERT_FILE).Match("Subject: CN = ").String()
		if err != nil {
			log.Fatalln("ERROR: here")
		}

		CERTNAME = strings.TrimSpace(strings.TrimPrefix(certName, "Subject: CN = "))

		cnPattern := regexp.MustCompile(`^[0-9a-z-]+\.[0-9a-z]{6,10}$`)
		if !cnPattern.MatchString(CERTNAME) {
			log.Fatalln("ERROR: cert subject CN invalid")
		}
	}

	steps := []Step{
		{Name: "installing required packages", Func: installPrerequisites},
		{Name: "configuring puppet", Func: configurePuppet},
		{Name: "node getting ready, sit back & relax \u2615 time !!", Func: puppet.RunPuppetWithRemoteLog},
	}

	// Add the subscription step only when subs is given in the args list
	if CERTNAME == "" || SUBSCRIPTION_LEVEL == "" {
		fmt.Println("Not adding subscription. Missing cert, key or subscription level. Contact Obmondo to add a subscription.")
	} else {
		newStep := Step{Name: "adding subscription, great job, asdad", Func: addSubscription}
		steps = append(steps, newStep)
	}

	spinnerStart(steps)
	script.Echo("Node setup completed.")
	if CERTNAME == "" || SUBSCRIPTION_LEVEL == "" {
		script.Echo("Please add the subscription on obmondo.com/server")
	}
	os.Exit(0)
}
