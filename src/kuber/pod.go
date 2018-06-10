package main

import (
	"fmt"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/tidwall/gjson"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Cost details
type Cost struct {
	totalCost   float64
	cpuCost     float64
	memoryCost  float64
	storageCost float64
}

// Pod Information
type Pod struct {
	name               string
	nodeName           string
	nodeCostPercentage float64
	cost               Cost
	pvcs               []*string
}

func getPodsForLabel(label string) []Pod {
	pods := []Pod{}
	command := fmt.Sprintf(getPodsByLabel, os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_KUBECONFIG"), label)
	bytes := executeCommand(command)
	json := string(bytes)
	items := gjson.Get(json, "items")

	items.ForEach(func(key, value gjson.Result) bool {
		name := value.Get("metadata.name")
		nodeName := value.Get("spec.nodeName")
		pod := Pod{name: name.Str, nodeName: nodeName.Str}

		podVolumes := []*string{}
		volumes := value.Get("spec.volumes")
		volumes.ForEach(func(volKey, volume gjson.Result) bool {
			pvc := volume.Get("persistentVolumeClaim.claimName")
			if pvc.Exists() {
				podVolumes = append(podVolumes, &pvc.Str)
			}
			return true
		})
		pod.pvcs = podVolumes
		pods = append(pods, pod)
		return true
	})
	return pods
}

func getPodsForLabelThroughClient(label string) []*Pod {
	vals := strings.Split(label, "=")
	if len(vals) != 2 {
		panic("Label should be of form key=val")
	}

	m := map[string]string{vals[0]: vals[1]}
	pods, err := ClientSetInstance.CoreV1().Pods("").List(metav1.ListOptions{LabelSelector: labels.SelectorFromSet(m).String()})
	if err != nil {
		panic(err.Error())
	}

	i := 0
	ps := []*Pod{}
	for i < len(pods.Items) {
		pod := pods.Items[i]
		p := Pod{}
		p.name = pod.GetObjectMeta().GetName()
		p.nodeName = pod.Spec.NodeName
		j := 0
		podVolumes := []*string{}
		for j < len(pod.Spec.Volumes) {
			vol := pod.Spec.Volumes[j]
			if vol.PersistentVolumeClaim != nil {
				podVolumes = append(podVolumes, &vol.PersistentVolumeClaim.ClaimName)
			}
			j++
		}
		p.pvcs = podVolumes
		ps = append(ps, &p)
		i++
	}
	return ps
}

func printPodsVerbose(pods []*Pod) {
	i := 0
	fmt.Printf("==Pods Cost Details==\n")
	totalCost := 0.0
	totalCPUCost := 0.0
	totalMemoryCost := 0.0
	totalStorageCost := 0.0
	for i <= len(pods)-1 {
		fmt.Printf("%-30s%s\n", "Pod Name:", pods[i].name)
		fmt.Printf("%-30s%s\n", "Node:", pods[i].nodeName)
		fmt.Printf("%-30s%.2f\n", "Pod Compute Cost Percentage:", pods[i].nodeCostPercentage*100.0)
		fmt.Printf("%-30s\n", "Persistent Volume Claims:")

		j := 0
		for j <= len(pods[i].pvcs)-1 {
			fmt.Printf("    %s\n", *pods[i].pvcs[j])
			j++
		}
		fmt.Printf("%-30s\n", "Cost:")
		fmt.Printf("    %-21s%f$\n", "Total Cost:", pods[i].cost.totalCost)
		fmt.Printf("    %-21s%f$\n", "CPU Cost:", pods[i].cost.cpuCost)
		fmt.Printf("    %-21s%f$\n", "Memory Cost:", pods[i].cost.memoryCost)
		fmt.Printf("    %-21s%f$\n", "Storage Cost:", pods[i].cost.storageCost)
		fmt.Printf("\n")

		totalCost += pods[i].cost.totalCost
		totalCPUCost += pods[i].cost.cpuCost
		totalMemoryCost += pods[i].cost.memoryCost
		totalStorageCost += pods[i].cost.storageCost
		i++
	}
	fmt.Printf("%-30s\n", "Total Cost Summary:")
	fmt.Printf("    %-21s%f$\n", "Total Cost:", totalCost)
	fmt.Printf("    %-21s%f$\n", "CPU Cost:", totalCPUCost)
	fmt.Printf("    %-21s%f$\n", "Memory Cost:", totalMemoryCost)
	fmt.Printf("    %-21s%f$\n", "Storage Cost:", totalStorageCost)
}

func printPodDetails(pods []Pod) {
	fmt.Println("===POD Details===")
	fmt.Println("POD Name \t\t\t\t\t Node Name")
	for _, value := range pods {
		fmt.Println(value.name + " \t" + value.nodeName)
	}
}