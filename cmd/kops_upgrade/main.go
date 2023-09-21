package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitfield/script"
)

const dirPath = "/resources"
const pvsPath = "/resources/pvs.txt"
const pdbsPath = "resources/pdbs.txt"

func main() {

	if len(os.Args) != 4 {
		fmt.Println("Command must have these arguements: <clusterName> <k8s Major Version> <k8s Minor Version>")
		os.Exit(1)
	}

	checkAwsCli()
	clustername := os.Args[1]
	majorVersion := os.Args[2]
	minorVersion := os.Args[3]

	checkDeprecatedAPIs(fmt.Sprintf("%s.%s", majorVersion, minorVersion), clustername)

	kopsBinaryName := setupKops(majorVersion, minorVersion)

	handlePVs()
	defer revertPVs()

	handlePDBs()
	defer revertPDBs()

	upgradeCluster(kopsBinaryName, clustername)

	// Post-upgrade
	_, err := script.Exec(fmt.Sprintf("%s get cluster %s -o yaml", kopsBinaryName, clustername)).Stdout()
	handleError(err)
}

func handleError(err error) {
	if err != nil {
		os.Exit(1)
	}
}

func handleStringError(errorMessage string, err error) {
	if err != nil {
		fmt.Printf("\nError: %s", errorMessage)
		os.Exit(1)
	}
}

// Upgrade the cluster to the same kops version major version, minor version and to latest patch
func upgradeCluster(kopsBinary, clusterName string) {
	commands := []string{
		fmt.Sprintf("%s upgrade cluster --name %s", kopsBinary, clusterName),
		fmt.Sprintf("%s upgrade cluster --name %s --yes", kopsBinary, clusterName),
		fmt.Sprintf("%s update cluster --name %s", kopsBinary, clusterName),
		fmt.Sprintf("%s update cluster --name %s --yes", kopsBinary, clusterName),
		fmt.Sprintf("%s rolling-update cluster --name %s", kopsBinary, clusterName),
		fmt.Sprintf("%s rolling-update cluster --name %s --yes", kopsBinary, clusterName),
	}

	for _, cmd := range commands {
		_, err := script.Exec(cmd).Stdout()
		handleError(err)
	}
}

// Downloading kops and setting it up
func setupKops(majorVersion, minorVersion string) string {
	fmt.Println("Setting up Kops")
	script.Exec("rm kops-linux-amd64")
	kopsURL := fmt.Sprintf("https://github.com/kubernetes/kops/releases/download/v%s.%s.0/kops-linux-amd64", majorVersion, minorVersion)
	fmt.Println("Downloading kops ...")
	_, err := script.Exec(fmt.Sprintf("wget %s", kopsURL)).String()
	if err != nil {
		fmt.Printf("Error downloading kops %s", err)
		os.Exit(1)
	}

	_, err = script.Exec("chmod u+x kops-linux-amd64").Stdout()
	handleError(err)

	versionOutput, err := script.Exec("./kops-linux-amd64 version").String()
	handleStringError(versionOutput, err)
	// Extracting the version from the format "Client version: 1.27.1 (git-v1.27.1)"
	splitStrings := strings.Split(versionOutput, " ")
	parsedVersion := strings.TrimSpace(splitStrings[2])

	newBinaryName := fmt.Sprintf("kops%s", parsedVersion)
	_, err = script.Exec(fmt.Sprintf("mv kops-linux-amd64 %s", newBinaryName)).Stdout()
	handleError(err)

	_, err = script.Exec(fmt.Sprintf("sudo mv %s /usr/local/bin", newBinaryName)).Stdout()
	handleError(err)
	return newBinaryName

}

// checkAwsCli will check if the cli is installed, if not install it and try to check if it is able to establish connection or not.
func checkAwsCli() {
	fmt.Println("Handling Aws Cli")
	// Check if AWS CLI is installed
	awsCLICheck, err := script.Exec("which aws").String()
	handleStringError(awsCLICheck, err)

	// If AWS CLI is not installed, check pip and then install AWS CLI
	if strings.TrimSpace(awsCLICheck) == "" {
		// Check if pip is installed
		pipCheck, err := script.Exec("which pip3").String()
		handleStringError(pipCheck, err)

		// If pip3 is not installed, install it
		if strings.TrimSpace(pipCheck) == "" {
			_, err = script.Exec("sudo apt install python3-pip -y").Stdout()
			handleError(err)
		}

		// Now install AWS CLI
		_, err = script.Exec("sudo pip3 install -U awscli").Stdout()
		handleError(err)
	}

	// Check AWS CLI connectivity
	awsTest, err := script.Exec("aws sts get-caller-identity").String()
	handleStringError(awsTest, err)
	fmt.Println("Successfully connected to AWS:", awsTest)
}

// HandlePVs will be patching the pvc which have delete recalim policy to retain
func handlePVs() {
	fmt.Println("Handling PVs...")
	ensureDirectory()

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}
	filePath := filepath.Join(cwd, pvsPath)

	// Check if file exists, if not create it
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		_, err := os.Create(filePath)
		if err != nil {
			log.Fatalf("Error creating file:%s", err)
		}
	}

	cmd := `kubectl get pv --all-namespaces -o=jsonpath='{range .items[?(@.spec.persistentVolumeReclaimPolicy=="Delete")]}{.metadata.name}{"\n"}{end}'`
	_, err = script.Exec(cmd).AppendFile(filePath)
	handleError(err)

	pvContents, err := script.File(filePath).String()
	handleStringError(pvContents, err)

	pvs := strings.Split(pvContents, "\n")
	for _, pv := range pvs {
		if pv == "" {
			continue
		}

		fmt.Println("Patching PV " + pv)
		_, err := script.Exec(fmt.Sprintf("kubectl patch pv %s -p '{\"spec\":{\"persistentVolumeReclaimPolicy\":\"Retain\"}}'", pv)).Stdout()
		handleError(err)
	}
}

// Revert PVs back to 'Delete' after upgrade
func revertPVs() {
	fmt.Println("Reverting PVCs...")
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}
	filePath := filepath.Join(cwd, pvsPath)

	pvContents, err := script.File(filePath).String()
	handleStringError(pvContents, err)

	pvs := strings.Split(pvContents, "\n")
	fmt.Println(pvs)
	for _, pv := range pvs {
		if pv == "" {
			continue
		}

		fmt.Println("Reverting PV " + pv)
		_, err := script.Exec(fmt.Sprintf("kubectl patch pv %s -p '{\"spec\":{\"persistentVolumeReclaimPolicy\":\"Delete\"}}'", pv)).Stdout()
		handleError(err)
	}
}

// Patch all pod disruption budgets to make minAvailable 0
func handlePDBs() {
	fmt.Println("Handling PDBs...")

	ensureDirectory()

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}
	filePath := filepath.Join(cwd, pdbsPath)

	// Check if file exists, if not create it
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		_, err := os.Create(filePath)
		if err != nil {
			log.Fatalf("Error creating file:%s", err)
		}
	}

	_, err = script.Exec("kubectl get pdb -o custom-columns=NAME:.metadata.name,MIN:.spec.minAvailable --no-headers").AppendFile(filePath)
	handleError(err)
	pdbContents, err := script.File(filePath).String()
	handleError(err)
	pdbLines := strings.Split(pdbContents, "\n")

	for _, pdbLine := range pdbLines {
		if pdbLine == "" {
			continue
		}
		parts := strings.Fields(pdbLine)
		pdbName := parts[0]

		fmt.Println("Patching PDB " + pdbName)
		_, err = script.Exec(fmt.Sprintf("kubectl patch pdb %s -p '{\"spec\":{\"minAvailable\":0}}'", pdbName)).Stdout()
		handleError(err)
	}
}

// Revert back the pod disruption budget to initial state
func revertPDBs() {
	fmt.Println("Reverting PDBs...")

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}
	filePath := filepath.Join(cwd, pdbsPath)

	pdbContents, err := script.File(filePath).String()
	handleStringError(pdbContents, err)

	pdbLines := strings.Split(pdbContents, "\n")
	for _, pdbLine := range pdbLines {
		if pdbLine == "" {
			continue
		}

		parts := strings.Fields(pdbLine)
		pdbName := parts[0]
		originalMinAvailable := parts[1]

		fmt.Println("Reverting PDB " + pdbName)
		_, err := script.Exec(fmt.Sprintf("kubectl patch pdb %s -p '{\"spec\":{\"minAvailable\":%s}}'", pdbName, originalMinAvailable)).Stdout()
		handleError(err)
	}
}

// checkDeprecatedAPIs store the deprecated if any into deprecated_apis.txt
func checkDeprecatedAPIs(k8sVersion, desiredClusterContext string) {
	fmt.Println("KUBECONFIG:", os.Getenv("KUBECONFIG"))

	// Check the current kubectl context
	currentContext, err := script.Exec("kubectl config current-context").String()
	handleStringError(currentContext, err)
	if strings.TrimSpace(currentContext) != desiredClusterContext {
		fmt.Printf("Current context (%s) does not match the desired context (%s).\n", currentContext, desiredClusterContext)
		os.Exit(1)
	}

	// Check for deprecated APIs
	kubepugOutput, err := script.Exec(fmt.Sprintf("kubepug --k8s-version=%s", k8sVersion)).String()
	handleStringError(kubepugOutput, err)
	if !strings.Contains(kubepugOutput, "No deprecated or deleted APIs found") {
		fmt.Println("Found deprecated APIs. Saving output to deprecated_apis.txt")
		_, err = script.File("deprecated_apis.txt").WriteFile(kubepugOutput)
		handleError(err)
	} else {
		fmt.Println("No deprecated APIs found.")
	}
}

// Ensuring that the directory exists
func ensureDirectory() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Failed to get current working directory: %s", err)
		os.Exit(1)
	}

	absDirPath := filepath.Join(cwd, dirPath)

	// Check if directory exists, if not create it
	if _, err := os.Stat(absDirPath); os.IsNotExist(err) {
		err = os.Mkdir(absDirPath, 0755)
		if err != nil {
			log.Fatalf("Failed to create directory: %v", err)
		}
	}
}
