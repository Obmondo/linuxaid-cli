package disk

import (
	"log"

	gpud "github.com/shirou/gopsutil/disk"
)

const (
	boot = 200000000
	root = 1500000000
)

var diskFreeSize = map[string]uint64{
	"/boot": boot,
	"/":     root,
}

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
		switch p.Mountpoint {
		case "/boot":
			// 200 MB
			if disk.Free <= diskFreeSize["/boot"] {
				log.Fatal("exiting, /boot has not enough free space ", disk.Free, " bytes. Need atleast 200MB free space")
			}
		case "/":
			// 1.5GB
			if disk.Free <= diskFreeSize["/"] {
				log.Fatal("exiting, / has not enough free space ", disk.Free, " bytes. Need atleast 1.5GB free space")
			}
			// no defaults, maybe I should have
		}
	}
}
