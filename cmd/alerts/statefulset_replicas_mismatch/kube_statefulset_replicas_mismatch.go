//go:build kube_statefulset_replicas_mismatch

package main

import (
	"errors"
	"fmt"
	"strconv"
)

var kubeStatefulSetReplicasMismatchAlertStepsInitialSteps = []scriptStep{
	{func() string { return "Stateful Set Replica Mismatch Resolver" }, headingPrint, "", "", false, "", false, false},
	{func() string { return "Run this script in another tab in your terminal." }, simplePrint, "", "", false, "", false, false},
	{func() string { return "Press Ctrl + C to exit this script at any time." }, simplePrint, "", "", false, "", false, false},

	{func() string { return "Enter the alert name" }, promptForStringInput, "alertName", "", false, "", false, false},

	{func() string { return "KubeStatefulSetReplicasMismatch,alertName" }, compareStrings, "", "This script is only applicable for KubeStatefulSetReplicasMismatch alerts.", false, "", false, false},

	{func() string { return "Enter the namespace" }, promptForStringInput, "namespace", "", false, "", false, false},

	{func() string { return "Enter the StatefulSet name" }, promptForStringInput, "statefulSet", "", false, "", false, false},

	{func() string {
		return fmt.Sprintf("Please check the status of all pods within namespace '%s'.", scriptInputsMap["namespace"])
	}, simplePrint, "", "", false, "", false, false},

	{func() string { return fmt.Sprintf("kubectl get pods -n %s", scriptInputsMap["namespace"]) }, commandPrint, "", "", false, "", false, false},

	{func() string { return "Are ALL pods fully ready (n/n) AND in running state?" }, promptForConfirmation, "", "If all pods are running and healthy, no further actions are needed.", true, "", false, false},

	{func() string {
		return fmt.Sprintf("Are ALL pods in the format %s-<n> ready and in the running state?", scriptInputsMap["statefulSet"])
	}, promptForConfirmation, "", "If all pods controled by the StatefulSet in question are running and healthy, but some other pods aren't, then this script may not be able to help you.", true, "", false, false},

	{func() string {
		return "For this script to help, the stateful set in question must NOT have pods with PVs attached to them"
	}, simplePrint, "", "", false, "", false, false},

	{func() string {
		return fmt.Sprintf(`kubectl get statefulset %s -o jsonpath='{.spec.volumeClaimTemplates}'`, scriptInputsMap["statefulSet"])
	}, commandPrint, "", "", false, "", false, false},

	{func() string { return "Did you get an EMPTY output?" }, promptForConfirmation, "", "No further actions can be suggested by this script as this script can only help with StatefulSet not matching expected replicas for pods that don't have PVs attached to them.", false, "", false, false},

	{func() string {
		return fmt.Sprintf("We now want to check if the PVs to which 'other' pods in the namespace (not pods of the form %s-<n>) are bound, allow writing.", scriptInputsMap["statefulSet"])
	}, simplePrint, "", "", false, "", false, false},

	{func() string {
		return fmt.Sprintf(`kubectl get pv -o=jsonpath='{range .items[?(@.status.phase=="Bound")]}{.metadata.name}{"\t"}{.spec.claimRef.namespace}{"\t"}{.spec.accessModes}{"\t"}{"\n"}{end}' | grep '%s'`, scriptInputsMap["namespace"])
	}, commandPrint, "", "", false, "", false, false},

	{func() string { return "Do all PVs allow writing (have an access mode like 'ReadWriteOnce')?" }, promptForConfirmation, "", "No further actions can be suggested by this script if a ReadOnly PV has been attached.", false, "", false, false},

	{func() string {
		return "Reaching here means that all the PVs attached to pods in this namespace allow writing. We now want to see if the pod that's failing is failing because it can't 'actually' write to a PV because that PV is not 'actually' allowing writes despite having access to write. We'll now exec into each pod which has a PV attached and check if you can actually write to the PV."
	}, simplePrint, "", "", false, "", false, false},

	{func() string { return "First, get each pod in running state." }, simplePrint, "", "", false, "", false, false},

	{func() string {
		return fmt.Sprintf(`kubectl get pods -n %s --field-selector=status.phase=Running`, scriptInputsMap["namespace"])
	}, commandPrint, "", "", false, "", false, false},

	{func() string { return "Enter the number of pods" }, promptForNumInput, "podsCountString", "", false, "", false, false},

	{func() string { return "Now, we will exec into each pod." }, simplePrint, "", "", false, "", false, false},
}

var kubeStatefulSetReplicasMismatchAlertLoopSteps = []scriptStep{
	{func() string { return "Enter the name of the pod" }, promptForStringInput, "currentPodname", "", false, "", false, false},

	{func() string {
		return fmt.Sprintf(`kubectl get pod %s -n %s -o jsonpath='{.spec.volumes[*].persistentVolumeClaim.claimName}'`, scriptInputsMap["currentPodname"], scriptInputsMap["namespace"])
	}, commandPrint, "", "", false, "", false, false},

	{func() string { return "Did you get any PVC?" }, promptForConfirmation, "foundPVC", "We only want to exec into pods that have a PV to see if we can create a test file. So, we'll move on to the next pod or exit if there's no next pod.", false, "", true, false},

	{func() string { return "Enter the name of the PVC you got" }, promptForStringInput, "currentPVC", "", false, "foundPVC", false, false},

	{func() string {
		return "We now want to find out the volume path for the PV where the pod writes and try writing to that volume."
	}, simplePrint, "", "", false, "foundPVC", false, false},

	{func() string {
		return fmt.Sprintf(`kubectl describe pod %s -n %s`, scriptInputsMap["currentPodname"], scriptInputsMap["namespace"])
	}, commandPrint, "", "", false, "foundPVC", false, false},

	{func() string {
		return fmt.Sprintf("Now, find the the name of the volume under 'Volumes' with 'claimName' %s.", scriptInputsMap["currentPVC"])
	}, simplePrint, "", "", false, "foundPVC", false, false},

	{func() string {
		return fmt.Sprintf("Were you able to find the volume with 'claimName' %s?", scriptInputsMap["currentPVC"])
	}, promptForConfirmation, "", "You must try to find the find the name under 'Volumes'. No further actions can be suggested by this script until you do that.", false, "foundPVC", false, false},

	{func() string { return "Enter the name of the volume you found" }, promptForStringInput, "currentVolume", "", false, "foundPVC", false, false},

	{func() string {
		return fmt.Sprintf("Now, find the 'Mounts' section and identify the mount path corresponding to %s", scriptInputsMap["currentVolume"])
	}, simplePrint, "", "", false, "foundPVC", false, false},

	{func() string { return "Were you able to identify the mount path?" }, promptForConfirmation, "", "You must try to find the identify the mount path corresponding to the volume name. No further actions can be suggested by this script until you do that.", false, "foundPVC", false, false},

	{func() string { return "Enter the exact mount path (starting with /) that you found" }, promptForStringInput, "currentMountPath", "", false, "foundPVC", false, false},

	{func() string { return "Now, we will exec into the pod and try to write in the mount path you found" }, simplePrint, "", "", false, "foundPVC", false, false},

	{func() string { return fmt.Sprintf(`kubectl exec -it %s /bin/bash`, scriptInputsMap["currentPodname"]) }, commandPrint, "", "", false, "foundPVC", false, false},

	{func() string { return "Were you able to exec into the pod?" }, promptForConfirmation, "", "It is important to exec into the pod to find out if you can write to it. No further actions can be suggested by this script until you do that.", false, "foundPVC", false, false},

	{func() string { return fmt.Sprintf(`cd %s`, scriptInputsMap["currentMountPath"]) }, commandPrint, "", "", false, "foundPVC", false, false},

	{func() string { return `pwd` }, commandPrint, "", "", false, "foundPVC", false, false},

	{func() string {
		return fmt.Sprintf("Is the present working directory %s?", scriptInputsMap["currentMountPath"])
	}, promptForConfirmation, "", "It is important to make sure the present working directory is the same as the mount path. No further actions can be suggested by this script until you do that.", false, "foundPVC", false, false},

	{func() string { return "Create a test file" }, simplePrint, "", "", false, "foundPVC", false, false},

	{func() string { return fmt.Sprintf(`touch testfile123`) }, commandPrint, "", "", false, "foundPVC", false, false},

	{func() string { return "Were you able to create a test file?" }, promptForConfirmation, "", "Exit the pod you 'execed' into", false, "foundPVC", false, true},

	{func() string {
		return "If you succeeded in creating a test file, then the PV linked to this pod does not have the issue this script looks at. We'll move on to the next pod or exit if there's no next pod."
	}, simplePrint, "", "", false, "foundPVC", false, false},

	{func() string { return "Exit the pod you 'execed' into" }, simplePrint, "", "", false, "foundPVC", false, false},
}

var kubeStatefulSetReplicasMismatchAlertStepsFinalSteps = []scriptStep{

	{func() string {
		return "podThatHasPVThatCannotBeWrittenTo,currentPodname,pvcForPVThatCannotBeWrittenTo,currentPVC,mountPathForPVThatCannotBeWrittenTo,currentMountPath"
	}, assignNewMapKeysFromExistingKeys, "", "Encountered an unexpected error while persisting details related to pod that has PV that cannot be written to", false, "", false, false},

	{func() string {
		return "We have found that a PV cannot be written to despite having the correct access rights. We will now attempt to detach and re-attach the PV by scaling the stateful set for the pod down and then back up again IF certain conditions are met."
	}, simplePrint, "", "", false, "", false, false},

	{func() string { return "Get the name of the StatefulSet that controls the pod." }, simplePrint, "", "", false, "", false, false},

	{func() string {
		return fmt.Sprintf(`kubectl get pod %s -n %s -o jsonpath='{.metadata.ownerReferences[?(@.kind=="StatefulSet")].name}'`, scriptInputsMap["podThatHasPVThatCannotBeWrittenTo"], scriptInputsMap["namespace"])
	}, commandPrint, "", "", false, "", false, false},

	{func() string { return "Enter the name of the StatefulSet you got" }, promptForStringInput, "statefulSetToBeScaled", "", false, "", false, false},

	{func() string {
		return "We will now check if the StatefulSet that controls the pod in question allows more than 1 replica."
	}, simplePrint, "", "", false, "", false, false},

	{func() string {
		return fmt.Sprintf(`kubectl get sts %s -n %s`, scriptInputsMap["statefulSetToBeScaled"], scriptInputsMap["namespace"])
	}, commandPrint, "", "", false, "", false, false},

	{func() string { return `Does the StatefulSet that controls the pod say 1/1 under 'Ready'?` }, promptForConfirmation, "", "No further actions can be suggested by this script.", false, "", false, false},

	{func() string {
		return "Next, we need to check the reclaim policy of the PV that you could not write to. It MUST be set to 'retain' before we can detach the PV."
	}, simplePrint, "", "", false, "", false, false},

	{func() string {
		return fmt.Sprintf(`kubectl get pvc %s -n %s -o jsonpath='{.spec.volumeName}'`, scriptInputsMap["pvcForPVThatCannotBeWrittenTo"], scriptInputsMap["namespace"])
	}, commandPrint, "", "", false, "", false, false},

	{func() string { return "Enter the name of the PV" }, promptForStringInput, "pvName", "", false, "", false, false},

	{func() string {
		return fmt.Sprintf(`kubectl get pv %s -n %s -o jsonpath='{.spec.persistentVolumeReclaimPolicy}'`, scriptInputsMap["pvName"], scriptInputsMap["namespace"])
	}, commandPrint, "", "", false, "", false, false},

	{func() string { return `Is the PV's reclaim policy set to something OTHER than 'retain' eg. 'delete'?` }, promptForConfirmation, "needToSetReclaimPolicyToRetain", "", false, "", false, false},

	{func() string {
		return fmt.Sprintf(`kubectl patch pv %s -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}`, scriptInputsMap["pvName"])
	}, commandPrint, "", "", false, "needToSetReclaimPolicyToRetain", false, false},

	{func() string {
		return fmt.Sprintf(`kubectl get pv %s -o jsonpath='{.spec.persistentVolumeReclaimPolicy}`, scriptInputsMap["pvName"])
	}, commandPrint, "", "", false, "needToSetReclaimPolicyToRetain", false, false},

	{func() string { return `Is the PV's reclaim policy now set to retain?` }, promptForConfirmation, "", "No further actions can be suggested by this script.", false, "needToSetReclaimPolicyToRetain", false, false},

	{func() string {
		return "We can now scale the StatefulSet that controls the pod in question down and then back up."
	}, simplePrint, "", "", false, "", false, false},

	{func() string {
		return fmt.Sprintf(`Please scale down the StatefulSet '%s' to 0 replicas`, scriptInputsMap["statefulSetToBeScaled"])
	}, simplePrint, "", "", false, "", false, false},

	{func() string {
		return fmt.Sprintf(`kubectl scale statefulsets %s --replicas=0 -n %s`, scriptInputsMap["statefulSetToBeScaled"], scriptInputsMap["namespace"])
	}, commandPrint, "", "", false, "", false, false},

	{func() string { return `Have you scaled down the StatefulSet to 0?` }, promptForConfirmation, "", "This script cannot suggest more actions till the StatefulSet is scaled down.", false, "", false, false},

	{func() string {
		return "Confirm that the pod controlled by the StatefulSet is NOT running after waiting for a short while."
	}, simplePrint, "", "", false, "", false, false},

	{func() string { return fmt.Sprintf(`kubectl get pods -n %s`, scriptInputsMap["namespace"]) }, commandPrint, "", "", false, "", false, false},

	{func() string { return `Is the pod linked to the stateful set running?` }, promptForConfirmation, "", "The pod linked to the stateful set must not run if the StatefulSet has been scaled down to 0. Exiting.", true, "", false, false},

	{func() string {
		return fmt.Sprintf("Now, scale up the StatefulSet '%s' back to 1", scriptInputsMap["statefulSetToBeScaled"])
	}, simplePrint, "", "", false, "", false, false},

	{func() string {
		return fmt.Sprintf(`kubectl scale statefulsets %s --replicas=1 -n %s`, scriptInputsMap["statefulSetToBeScaled"], scriptInputsMap["namespace"])
	}, commandPrint, "", "", false, "", false, false},

	{func() string { return "Check if the pod attached to the PV is up and running again." }, simplePrint, "", "", false, "", false, false},

	{func() string { return fmt.Sprintf(`kubectl get pods -n %s`, scriptInputsMap["namespace"]) }, commandPrint, "", "", false, "", false, false},

	{func() string { return `Is the pod attached to the PV been scaled up and is it running again?` }, promptForConfirmation, "", "Please ensure that the StatefulSet is scaled up and pods are running. Exiting.", false, "", false, false},

	{func() string {
		return "Let's try to exec into the pod that we just recreated and try to write to the PV that we could not write to earlier."
	}, simplePrint, "", "", false, "", false, false},

	{func() string {
		return fmt.Sprintf(`kubectl exec -it %s /bin/bash`, scriptInputsMap["podThatHasPVThatCannotBeWrittenTo"])
	}, commandPrint, "", "", false, "", false, false},

	{func() string { return `Were you able to exec into the pod?` }, promptForConfirmation, "", "It is important to exec into the pod to find out if you can write to it. No further actions can be suggested by this script.", false, "", false, false},

	{func() string { return fmt.Sprintf(`cd %s`, scriptInputsMap["mountPathForPVThatCannotBeWrittenTo"]) }, commandPrint, "", "", false, "", false, false},

	{func() string { return fmt.Sprintf(`pwd`) }, commandPrint, "", "", false, "", false, false},

	{func() string {
		return fmt.Sprintf("Is the present working directory %s?", scriptInputsMap["mountPathForPVThatCannotBeWrittenTo"])
	}, promptForConfirmation, "", "It is important to make sure the present working directory is the same as the mount path. No further actions can be suggested by this script until you do that.", false, "", false, false},

	{func() string { return "Create a test file" }, simplePrint, "", "", false, "", false, false},

	{func() string { return fmt.Sprintf(`touch testfile123`) }, commandPrint, "", "", false, "", false, false},

	{func() string { return `Were you able to create test file?` }, promptForConfirmation, "", "This script can't help you with the issue as you should have been able to create a test file based on the actions taken so far. Do not forget to the exit the pod you just 'execed' into.", false, "", false, false},

	{func() string { return "Exit the pod you execed into." }, simplePrint, "", "", false, "", false, false},

	{func() string {
		return fmt.Sprintf("If any pods of the form %s-<n> were not running previously, they will need to be deleted to be recreated properly.", scriptInputsMap["statefulSet"])
	}, simplePrint, "", "", false, "", false, false},

	{func() string { return fmt.Sprintf(`kubectl get pods -n %s `, scriptInputsMap["namespace"]) }, commandPrint, "", "", false, "", false, false},

	{func() string {
		return fmt.Sprintf("Enter the name of the pod of the form %s-<n> to delete", scriptInputsMap["statefulSet"])
	}, promptForStringInput, "currentPodname", "", false, "", false, false},

	{func() string {
		return fmt.Sprintf(`kubectl delete pod %s -n %s`, scriptInputsMap["currentPodname"], scriptInputsMap["namespace"])
	}, commandPrint, "", "", false, "", false, false},

	{func() string {
		return fmt.Sprintf("Do this for all pods of the form %s-<n>", scriptInputsMap["statefulSet"])
	}, simplePrint, "", "", false, "", false, false},

	{func() string { return "Verify all pods in the namespace are healthy and running." }, simplePrint, "", "", false, "", false, false},

	{func() string { return fmt.Sprintf(`kubectl get pods -n %s`, scriptInputsMap["namespace"]) }, commandPrint, "", "", false, "", false, false},

	{func() string { return `Are all the pods healthy and running as expected?` }, promptForConfirmation, "", "No further actions can be suggested by this script.", false, "", false, false},

	{func() string { return "Congratulations, the alert can be resolved now." }, simplePrint, "", "", false, "", false, false},
}

func main() {

	if err := executeSteps(kubeStatefulSetReplicasMismatchAlertStepsInitialSteps); err != nil {
		return
	}
	podsCount, _ := strconv.Atoi(scriptInputsMap["podsCountString"])

	var err error
	for i := 0; i < podsCount; i++ {
		if err = executeSteps(kubeStatefulSetReplicasMismatchAlertLoopSteps); err != nil {
			if errors.Is(err, errMoveOn) {
				continue
			}
			break
		}
	}
	if !errors.Is(err, errBreak) {
		return
	}

	if err := executeSteps(kubeStatefulSetReplicasMismatchAlertStepsFinalSteps); err != nil {
		return
	}
}
