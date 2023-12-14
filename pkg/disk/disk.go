package disk

import (
	"fmt"
	"log"

	gpud "github.com/shirou/gopsutil/disk"
)

const (
	diskSpace = 00
)

func listPartitions() ([]gpud.PartitionStat, error) {
	// Only returns physical devices only (e.g. hard disks, cd-rom drives, USB keys)
	allPartitions, err := gpud.Partitions(false)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return allPartitions, nil
}

func CheckDiskSize() {
	partitions, err := listPartitions()
	if err != nil {
		log.Fatal("Failed to fetch partitions: ", err)
	}

	for _, p := range partitions {
		disk, err := gpud.Usage(p.Mountpoint)
		if err != nil {
			log.Println("failed")
		}

		if p.Mountpoint == "/boot" || p.Mountpoint == "/" {
			if disk.Free == diskSpace {
				errMsg := fmt.Sprintf("%s has %v bytes of space left, exiting", p.Mountpoint, disk.Free)
				log.Println(errMsg)
			}
		}
	}
}
