package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bitfield/script"

	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const dirPath = "/resources"
const pvsPath = "/resources/pvs.txt"
const pdbsPath = "resources/pdbs.txt"
const deprecated_apis = "/resources/depreciated_apis.txt"

func main() {

	if len(os.Args) < 4 || len(os.Args) > 5 {
		fmt.Println("Command must have these arguments: <clusterName> <k8s Major Version> <k8s Minor Version> [handlePdb]")
		fmt.Println("handlePdb is optional. If you want to handle PDB as well, add the 4th argument as 'handlePdb'")
		os.Exit(1)
	}

	clustername := os.Args[1]
	majorVersion := os.Args[2]
	minorVersion := os.Args[3]

	// Basic checks
	if clustername == "" {
		fmt.Println("Clustername cannot be an empty string.")
		os.Exit(1)
	}

	// Checking major in between 0 and 1
	if !isValidVersion(majorVersion, 0, 1) {
		os.Exit(1)
	}

	// Checking minor in between 20 30
	if !isValidVersion(minorVersion, 20, 30) {
		os.Exit(1)
	}

	checkAwsCli()

	checkDeprecatedAPIs(fmt.Sprintf("%s.%s", majorVersion, minorVersion), clustername)

	checkKopsVersion(majorVersion, minorVersion)

	handlePVs()
	defer revertPVs()

	if len(os.Args) == 5 {
		hadlePDB := os.Args[4]
		if hadlePDB == "handlePdb" {
			handlePDBs()
			defer revertPDBs()
		}
	}

	upgradeCluster(clustername)

	// Post-upgrade
	_, err := script.Exec(fmt.Sprintf("kops get cluster %s -o yaml", clustername)).Stdout()
	handleError(err)
}

func handleError(err error) {
	if err != nil {
		os.Exit(1)
	}
}

func isValidVersion(ver string, min int, max int) bool {
	v, err := strconv.Atoi(ver)
	if err != nil {
		fmt.Println("Version should be a valid integer.")
		return false
	}
	if v < min || v > max {
		fmt.Printf("Version should be between %d and %d.\n", min, max)
		return false
	}
	return true
}

func handleStringError(errorMessage string, err error) {
	if err != nil {
		fmt.Printf("\nError: %s", errorMessage)
		os.Exit(1)
	}
}

// promptUser prompts the user with a given message and expects 'yes' as confirmation to proceed.
func promptUser(message string) {
	fmt.Println(message)
	fmt.Print("Do you want to proceed? (yes/no): ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to read input: %v", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "yes" {
		fmt.Println("Operation aborted.")
		os.Exit(1)
	}
}

// Upgrade the cluster to the same kops version major version, minor version and to latest patch
func upgradeCluster(clusterName string) {
	commands := []string{
		fmt.Sprintf("kops upgrade cluster --name %s", clusterName),
		fmt.Sprintf("kops upgrade cluster --name %s --yes", clusterName),
		fmt.Sprintf("kops update cluster --name %s", clusterName),
		fmt.Sprintf("kops update cluster --name %s --yes", clusterName),
		fmt.Sprintf("kops rolling-update cluster --name %s", clusterName),
		fmt.Sprintf("kops rolling-update cluster --name %s --yes", clusterName),
	}

	for _, cmd := range commands {
		if strings.Contains(cmd, "--yes") {
			promptUser(fmt.Sprintf("About to execute: %s", cmd))
		}
		_, err := script.Exec(cmd).Stdout()
		handleError(err)
	}
}

// Downloading kops and setting it up
func checkKopsVersion(majorVersion, minorVersion string) {
	fmt.Println("Checking Kops version...")

	// Check if kops is installed
	kopsCheck, err := script.Exec("which kops").String()
	handleStringError(kopsCheck, err)

	if strings.TrimSpace(kopsCheck) == "" {
		fmt.Println("Kops is not installed. Please install Kops and set it to path.")
		os.Exit(1)
	}

	// Get kops version
	versionOutput, err := script.Exec("kops version").String()
	handleStringError(versionOutput, err)

	// Extracting the version from the format "Client version: 1.27.1 (git-v1.27.1)"
	splitStrings := strings.Split(versionOutput, " ")
	currentVersion := strings.TrimSpace(splitStrings[1])
	splitVersionParts := strings.Split(currentVersion, ".")
	if len(splitVersionParts) < 3 {
		fmt.Println("Error parsing kops version.")
		os.Exit(1)
	}

	// Check if major and minor versions match the desired ones
	if splitVersionParts[0] != majorVersion || splitVersionParts[1] != minorVersion {
		fmt.Printf("Currently installed kops version is %s. Please install version %s.%s for upgradation.\n", currentVersion, majorVersion, minorVersion)
		os.Exit(1)
	}

	fmt.Println("Correct version of Kops is installed!")
}

// checkAwsCli will check if the cli is installed, if not install it and try to check if it is able to establish connection or not.
func checkAwsCli() {
	fmt.Println("Checking AWS CLI installation...")

	// Check if AWS CLI is installed
	awsCLICheck, err := script.Exec("which aws").String()
	handleStringError(awsCLICheck, err)

	// If AWS CLI is not installed, exit with a message
	if strings.TrimSpace(awsCLICheck) == "" {
		fmt.Println("AWS CLI not found. Please install AWS CLI.")
		os.Exit(1)
	}

	// Check AWS CLI connectivity
	awsTest, err := script.Exec("aws sts get-caller-identity").String()

	// If there's an error, it likely means AWS CLI isn't configured properly
	if err != nil {
		fmt.Println("Unable to connect to AWS. Please configure your AWS CLI.")
		os.Exit(1)
	}

	fmt.Println("Successfully connected to AWS:", awsTest)
}

// HandlePVs will be patching the pvc which have delete recalim policy to retain
func handlePVs() {
	fmt.Println("Handling PVs...")

	// Create Kubernetes client
	client, err := createK8sClient()
	handleError(err)

	pvs, err := client.CoreV1().PersistentVolumes().List(context.TODO(), metav1.ListOptions{})
	handleError(err)

	var pvNamesToDelete []string
	for _, pv := range pvs.Items {
		if pv.Spec.PersistentVolumeReclaimPolicy == v1.PersistentVolumeReclaimDelete {
			pvNamesToDelete = append(pvNamesToDelete, pv.Name)
		}
	}

	// Write PV names to the file
	ensureDirectory()
	cwd, _ := os.Getwd()
	filePath := filepath.Join(cwd, pvsPath)
	file, err := os.Create(filePath)
	handleError(err)
	defer file.Close()
	for _, name := range pvNamesToDelete {
		file.WriteString(name + "\n")
	}

	// Patch PVs
	for _, pvName := range pvNamesToDelete {
		fmt.Println("Patching PV " + pvName)

		// Fetch the PersistentVolume with the specified name
		pvToUpdate, err := client.CoreV1().PersistentVolumes().Get(context.TODO(), pvName, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("failed to get PersistentVolume %s: %v", pvName, err)
			os.Exit(1)
		}

		// Update the reclaim policy
		pvToUpdate.Spec.PersistentVolumeReclaimPolicy = v1.PersistentVolumeReclaimRetain

		// Submit the update back to the API server
		_, err = client.CoreV1().PersistentVolumes().Update(context.TODO(), pvToUpdate, metav1.UpdateOptions{})
		if err != nil {
			fmt.Printf("failed to update PersistentVolume %s: %v", pvName, err)
			os.Exit(1)
		}
	}
}

// Revert PVs back to 'Delete' after upgrade
func revertPVs() {
	fmt.Println("Reverting PVs...")
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}
	filePath := filepath.Join(cwd, pvsPath)

	pvContents, err := script.File(filePath).String()
	handleStringError(pvContents, err)

	pvs := strings.Split(pvContents, "\n")
	fmt.Println(pvs)

	// Create Kubernetes client
	client, err := createK8sClient()
	handleError(err)

	for _, pvName := range pvs {
		if pvName == "" {
			continue
		}

		fmt.Println("Reverting PV " + pvName)

		// Fetch the PersistentVolume with the specified name
		pvToUpdate, err := client.CoreV1().PersistentVolumes().Get(context.TODO(), pvName, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("failed to get PersistentVolume %s: %v", pvName, err)
			os.Exit(1)
		}

		// Update the reclaim policy
		pvToUpdate.Spec.PersistentVolumeReclaimPolicy = v1.PersistentVolumeReclaimDelete

		_, err = client.CoreV1().PersistentVolumes().Update(context.TODO(), pvToUpdate, metav1.UpdateOptions{})
		if err != nil {
			fmt.Printf("failed to update PersistentVolume %s: %v", pvName, err)
			os.Exit(1)
		}
	}
}

// Patch all pod disruption budgets to make minAvailable 0
func handlePDBs() {
	fmt.Println("Handling PDBs...")
	ensureDirectory()

	// Check if file exists, if not create it
	cwd, err := os.Getwd()
	handleError(err)
	filePath := filepath.Join(cwd, pdbsPath)
	file, err := os.Create(filePath)
	handleError(err)
	defer file.Close()

	// Create Kubernetes client
	client, err := createK8sClient()
	handleError(err)

	// Fetch PDBs
	pdbs, err := client.PolicyV1().PodDisruptionBudgets("").List(context.TODO(), metav1.ListOptions{})
	handleError(err)

	for _, pdb := range pdbs.Items {
		pdbName := pdb.ObjectMeta.Name
		minAvailable := pdb.Spec.MinAvailable
		pdbNamespcace := pdb.ObjectMeta.Namespace

		// Store original pdb details in file
		_, err := file.WriteString(fmt.Sprintf("%s %s %v\n", pdbName, pdbNamespcace, minAvailable))
		if err != nil {
			log.Fatalf("Failed to write to file: %v", err)
		}

		fmt.Println("Patching PDB " + pdbName)
		m := intstr.FromInt(0)
		pdb.Spec.MinAvailable = &m
		_, err = client.PolicyV1().PodDisruptionBudgets(pdb.Namespace).Update(context.TODO(), &pdb, metav1.UpdateOptions{})
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

	// Read the stored original pdb details from file
	pdbContents, err := script.File(filePath).String()
	handleError(err)

	pdbLines := strings.Split(pdbContents, "\n")

	// Create Kubernetes client
	client, err := createK8sClient()
	handleError(err)

	for _, pdbLine := range pdbLines {
		if pdbLine == "" {
			continue
		}

		parts := strings.Fields(pdbLine)
		pdbName := parts[0]
		pdbNamespace := parts[1]
		originalMinAvailable, err := strconv.Atoi(parts[2])
		handleError(err)

		fmt.Println("Reverting PDB " + pdbName)

		// Fetch the specific PDB
		pdb, err := client.PolicyV1().PodDisruptionBudgets(pdbNamespace).Get(context.TODO(), pdbName, metav1.GetOptions{})
		handleError(err)

		// Set the PDB's minAvailable back to its original value and update
		m := intstr.FromInt(originalMinAvailable)
		pdb.Spec.MinAvailable = &m
		_, err = client.PolicyV1().PodDisruptionBudgets(pdb.Namespace).Update(context.TODO(), pdb, metav1.UpdateOptions{})
		handleError(err)
	}
}

// checkDeprecatedAPIs store the deprecated if any into deprecated_apis.txt
func checkDeprecatedAPIs(k8sVersion, desiredClusterContext string) {
	kubeconfig := os.Getenv("KUBECONFIG")

	if kubeconfig == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		} else {
			fmt.Println("could not find home directory")
			os.Exit(1)
		}
	}

	// Check if kubepug is installed
	kubepugCheck, err := script.Exec("which kubepug").String()
	handleStringError(kubepugCheck, err)
	if strings.TrimSpace(kubepugCheck) == "" {
		fmt.Println("Kubepug is not installed. Please install it to continue.")
		os.Exit(1)
	}

	// Check the current kubectl context
	currentContext, err := script.Exec(fmt.Sprintf("kubectl config current-context --kubeconfig=%s", kubeconfig)).String()
	handleStringError(currentContext, err)
	if strings.TrimSpace(currentContext) != desiredClusterContext {
		fmt.Printf("Current context %s does not match the desired context %s.\n", currentContext, desiredClusterContext)
		os.Exit(1)
	}

	// Check for deprecated APIs
	kubepugOutput, err := script.Exec(fmt.Sprintf("kubepug --k8s-version=v%s --kubeconfig=%s", k8sVersion, kubeconfig)).String()
	handleStringError(kubepugOutput, err)
	if !strings.Contains(kubepugOutput, "No deprecated or deleted APIs found") {
		fmt.Println("Found deprecated APIs. Saving output to deprecated_apis.txt")

		// Check if file exists, if not create it
		cwd, err := os.Getwd()
		handleError(err)
		filePath := filepath.Join(cwd, deprecated_apis)
		file, err := os.Create(filePath)
		handleError(err)
		defer file.Close()

		_, err = script.File(filePath).WriteFile(kubepugOutput)
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

func createK8sClient() (*kubernetes.Clientset, error) {
	kubeconfig := os.Getenv("KUBECONFIG")

	if kubeconfig == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		} else {
			return nil, fmt.Errorf("could not find home directory")
		}
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}
