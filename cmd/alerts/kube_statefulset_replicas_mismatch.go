//go:build kube_statefulset_replicas_mismatch

package main

import (
	"fmt"
	"strconv"
)

const kubeStatefulSetReplicasMismatchAlert = "KubeStatefulSetReplicasMismatch"

func main() {
	// Explain what the script does
	headingPrint("Stateful Set Replica Mismatch Resolver")
	fmt.Println("Run this script in another tab in your terminal, not the tab where you run Kubernetes commands.")
	fmt.Println("Press Ctrl + C to exit this script at any time.")

	// Validate alert name
	alertName := promptForStringInput("Enter the alert name")
	if alertName != kubeStatefulSetReplicasMismatchAlert {
		fmt.Printf("This script is only applicable for '%s' alerts.\n", kubeStatefulSetReplicasMismatchAlert)
		return
	}

	// Collect basic info that's needed
	namespace := promptForStringInput("Enter the namespace")
	statefulSet := promptForStringInput("Enter the stateful set name")

	// Execution steps begin
	fmt.Printf("Please check the status of all pods within namespace '%s'.\n", namespace)
	fmt.Println("Run the following command:")
	commandPrint(fmt.Sprintf("kubectl get pods -n %s\n", namespace))

	if promptForConfirmation("Are ALL pods fully ready (n/n) AND in running state?") {
		fmt.Println("If all pods are running and healthy, no further actions are needed.")
		return
	}

	if promptForConfirmation(fmt.Sprintf("Are ALL pods in the format %s-<n> ready and in the running state?", statefulSet)) {
		fmt.Println(fmt.Sprintf("If all pods whose name is in the format %s-<n> are running and healthy, but some other pods aren't, then this script may not be able to help you.", statefulSet))
		return
	}

	fmt.Println("For this script to help, the stateful set in question must NOT have pods with PVs attached to them")
	fmt.Println("Run the command:")
	commandPrint(fmt.Sprintf(`kubectl get statefulset %s -o jsonpath='{.spec.volumeClaimTemplates}'`, statefulSet))
	if !promptForConfirmation("Did you get an EMPTY output?") {
		fmt.Println("No further actions can be suggested by this script as this script can only help with StatefulSet not matching expected replicas for pods that don't have PVs attached to them.")
		return
	}

	fmt.Println(fmt.Sprintf("We now want to check if the PVs to which 'other' pods in the namespace (not pods of the form %s-<n>) are bound, allow writing. Run the following command:", statefulSet))

	commandPrint(fmt.Sprintf(`kubectl get pv -o=jsonpath='{range .items[?(@.status.phase=="Bound")]}{.metadata.name}{"\t"}{.spec.claimRef.namespace}{"\t"}{.spec.accessModes}{"\t"}{"\n"}{end}' | grep '%s'`, namespace))

	if !promptForConfirmation("Do all PVs allow writing (have an access mode like 'ReadWriteOnce')?") {
		fmt.Println("No further actions can be suggested by this script if a ReadOnly PV has been attached.")
		return
	}

	fmt.Println("Reaching here means that all the PVs attached to pods in this namespace allow writing. We now want to see if the pod that's failing is failing because it can't 'actually' write to a PV because that PV is not 'actually' allowing writes despite having access to write. We'll now exec into each pod which has a PV attached and check if you can actually write to the PV.")
	fmt.Println("First, get each pod in running state.")
	fmt.Println("Run the command:")
	commandPrint(fmt.Sprintf(`kubectl get pods -n %s --field-selector=status.phase=Running`, namespace))
	podsCountString := promptForNumInput("Enter the number of pods")
	podsCount, _ := strconv.Atoi(podsCountString)

	fmt.Println("Now, we will exec into each pod.")
	var (
		podThatHasPVThatCannotBeWrittenTo   string
		pvcForPVThatCannotBeWrittenTo       string
		mountPathForPVThatCannotBeWrittenTo string
	)
	for i := 0; i < podsCount; i++ {
		podname := promptForStringInput("Enter the name of the pod")

		fmt.Println("Run the command:")
		commandPrint(fmt.Sprintf(`kubectl get pod %s -n %s -o jsonpath='{.spec.volumes[*].persistentVolumeClaim.claimName}'`, podname, namespace))
		if !promptForConfirmation("Did you get any PVC?") {
			fmt.Println("We only want to exec into pods that have a PV to see if we can create a test file. So, we'll move on to the next pod.")
			continue
		}
		pvcName := promptForStringInput("Enter the name of the PVC you got")

		fmt.Println("We now want to find out the volume path for the PV where the pod writes and try writing to that volume.")
		fmt.Println("Run the command:")
		commandPrint(fmt.Sprintf(`kubectl describe pod %s -n %s`, podname, namespace))
		fmt.Println(fmt.Sprintf("Now, find the the name of the volume under 'Volumes' with 'claimName' %s.", pvcName))
		for !promptForConfirmation(fmt.Sprintf("Were you able to find the volume with 'claimName' %s?", pvcName)) {
			fmt.Println("You must try to find the find the name under 'Volumes'. No further actions can be suggested by this script until you do that.")
		}

		volumeName := promptForStringInput("Enter the name of the volume you found")

		fmt.Println(fmt.Sprintf("Now, find the 'Mounts' section and identify the mount path corresponding to %s", volumeName))
		for !promptForConfirmation("Were you able to identify the mount path?") {
			fmt.Println("You must try to find the identify the mount path corresponding to the volume name. No further actions can be suggested by this script until you do that.")
		}
		mountPath := promptForStringInput("Enter the exact mount path (starting with /) that you found")
		fmt.Println("Now, we will exec into the pod and try to write in the mount path you found")
		fmt.Println("Run the command:")

		commandPrint(fmt.Sprintf(`kubectl exec -it %s /bin/bash`, podname))
		for !promptForConfirmation("Were you able to exec into the pod?") {
			fmt.Println("It is important to exec into the pod to find out if you can write to it. No further actions can be suggested by this script until you do that.")

		}
		fmt.Println("Run the command:")
		commandPrint(fmt.Sprintf(`cd %s`, mountPath))
		fmt.Println("Run the command:")
		commandPrint(fmt.Sprintf(`pwd`))
		for !promptForConfirmation(fmt.Sprintf("Is the present working directory %s?", mountPath)) {
			fmt.Println("It is important to make sure the present working directory is the same as the mount path. No further actions can be suggested by this script until you do that.")
		}
		fmt.Println(`Create a test file`)
		fmt.Println("Run the command:")
		commandPrint(`touch testfile123`)
		if promptForConfirmation("Were you able to create test file)?") {
			fmt.Println("If you succeeded in creating a test file, then the PV linked to this pod does not have the issue this script looks at. We'll move on to the next pod.")
		} else {
			podThatHasPVThatCannotBeWrittenTo = podname
			pvcForPVThatCannotBeWrittenTo = pvcName
			mountPathForPVThatCannotBeWrittenTo = mountPath
			break
		}
		fmt.Println("Exit the pod you 'execed' into")
		fmt.Println("Run the command:")
		commandPrint(`exit`)

	}

	if len(podThatHasPVThatCannotBeWrittenTo) == 0 {
		fmt.Println("No further actions can be suggested by this script.")
		return
	}

	fmt.Println("We have found that a PV cannot be written to despite having the correct access rights. We will now attempt to detach and re-attach the PV by scaling the stateful set for the pod down and then back up again IF certain conditions are met.")
	fmt.Println("Get the name of the StatefulSet that controls the pod")
	fmt.Println("Run the command:")
	commandPrint(fmt.Sprintf(`kubectl get pod %s -n %s -o jsonpath='{.metadata.ownerReferences[?(@.kind=="StatefulSet")].name}'
	`, podThatHasPVThatCannotBeWrittenTo, namespace))
	statefulSetName := promptForStringInput("Enter the name of the StatefulSet you got")

	fmt.Println("We will now check if the StatefulSet that controls the pod in question allows more than 1 replica.")
	fmt.Println("Run the command:")
	commandPrint(fmt.Sprintf(`kubectl get sts %s -n %s`, statefulSet, namespace))
	if !promptForConfirmation("Does the StatefulSet that controls the pod say 1/1 under 'Ready'?") {
		fmt.Println("No further actions can be suggested by this script.")
		return
	}
	fmt.Println("Next, we need to check the reclaim policy of the PV that you could not write to. It MUST be set to 'retain' before we can detach the PV.")
	fmt.Println("Run the command:")
	commandPrint(fmt.Sprintf(`kubectl get pvc %s -n %s -o jsonpath='{.spec.volumeName}'
	`, pvcForPVThatCannotBeWrittenTo, namespace))

	pvName := promptForStringInput("Enter the name of the PV")

	fmt.Println("Run the command:")
	commandPrint(fmt.Sprintf(`kubectl get pv %s -o jsonpath='{.spec.persistentVolumeReclaimPolicy}'
	`, pvName))

	if !promptForConfirmation("Is the PV's reclaim policy set to retain?") {
		fmt.Println("Run the command:")
		commandPrint(fmt.Sprintf(`kubectl patch pv %s -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}`, pvName))

		fmt.Println("Run the command:")
		commandPrint(fmt.Sprintf(`kubectl get pv %s -o jsonpath='{.spec.persistentVolumeReclaimPolicy}'
	`, pvName))
		if !promptForConfirmation("Is the PV's reclaim policy now set to retain?") {
			fmt.Println("No further actions can be suggested by this script.")
			return
		}
	}

	fmt.Println("We can now scale the StatefulSet that controls the pod in question down and then back up")

	fmt.Printf("Please scale down the StatefulSet '%s' to 0 replicas. Run the following command:\n", statefulSetName)
	commandPrint(fmt.Sprintf("kubectl scale statefulsets %s --replicas=0 -n %s\n", statefulSetName, namespace))

	if !promptForConfirmation("Have you scaled down the StatefulSet to 0?") {
		fmt.Println("Please scale down the StatefulSet before continuing.")
		return
	}

	fmt.Println("Confirm that the pod controlled by the StatefulSet is NOT running")

	fmt.Println("Run the command:")
	commandPrint(fmt.Sprintf(`kubectl get pods -n %s`, namespace))

	for promptForConfirmation("Is the pod linked to the stateful set running?") {
		fmt.Println("Please scale down the StatefulSet before continuing.")

	}
	fmt.Printf("Now, scale up the StatefulSet '%s' back to 1. Run the following command:\n", statefulSetName)
	commandPrint(fmt.Sprintf("kubectl scale statefulsets %s --replicas=1 -n %s\n", statefulSetName, namespace))

	fmt.Println("Check if the pod attached to the PV is up and running again")
	fmt.Println("Run the command:")
	commandPrint(fmt.Sprintf(`kubectl get pods -n %s`, namespace))

	for !promptForConfirmation("Have the pods attached to the PV been scaled up and are they running again?") {
		fmt.Println("Please ensure that the StatefulSet is scaled up and pods are running before continuing.")
		return
	}

	fmt.Println("Let's try to exec into the pod that we just recreated and try to write to the PV that we could not write to earlier")
	commandPrint(fmt.Sprintf(`kubectl exec -it %s /bin/bash`, podThatHasPVThatCannotBeWrittenTo))
	for !promptForConfirmation("Were you able to exec into the pod)?") {
		fmt.Println("It is important to exec into the pod to find out if you can write to it. No further actions can be suggested by this script.")

	}
	fmt.Println("Run the command:")
	commandPrint(fmt.Sprintf(`cd %s`, mountPathForPVThatCannotBeWrittenTo))
	fmt.Println("Run the command:")
	commandPrint(fmt.Sprintf(`pwd`))
	for !promptForConfirmation(fmt.Sprintf("Is the present working directory %s?", mountPathForPVThatCannotBeWrittenTo)) {
		fmt.Println("It is important to make sure the present working directory is the same as the mount path. No further actions can be suggested by this script until you do that.")
	}
	fmt.Println(`Create a test file`)
	fmt.Println("Run the command:")
	commandPrint(`touch testfile123`)
	if promptForConfirmation("Were you able to create test file)?") {
		fmt.Println("If you succeeded in creating a test file, then the PV linked to this pod can now be written to - which is great because it couldn't be written to earlier.")
	}
	fmt.Println("Exit the pod you execed into")
	fmt.Println("Run the command:")
	commandPrint(`exit`)

	fmt.Println(fmt.Sprintf("If any pods of the form %s-<n> were not running previously, they will need to be deleted to be recreated properly.", statefulSet))
	fmt.Println("Run the command:")
	commandPrint(fmt.Sprintf(`kubectl get pods -n %s `, namespace))
	podname := promptForStringInput(fmt.Sprintf("Enter the name of the pod of the form %s-<n> to delete", statefulSet))

	fmt.Println("Run the command:")
	fmt.Printf("kubectl delete pod %s -n %s\n", podname, namespace)

	fmt.Println(fmt.Sprintf("Do this for all pods of the form %s-<n>", statefulSet))
	fmt.Println("Verify all pods in the namespace are healthy and running. Run the following command:")
	fmt.Printf("kubectl get pods -n %s\n", namespace)

	if !promptForConfirmation("Are all the pods healthy and running as expected?") {
		fmt.Println("No further actions can be suggested by this script.")
		return
	}

	fmt.Println("Congratulations, the alert can be resolved now.")
}
