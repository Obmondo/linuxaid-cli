package disk

import (
	"errors"
	"fmt"
	"log/slog"

	gpud "github.com/shirou/gopsutil/disk"
)

const (
	boot = 10000000  // 10Mb
	root = 100000000 // 100Mb
)

var diskFreeSize = map[string]uint64{
	"/boot": boot,
	"/":     root,
}

func listPartitions() ([]gpud.PartitionStat, error) {
	// Only returns physical devices only (e.g. hard disks, cd-rom drives, USB keys)
	allPartitions, err := gpud.Partitions(false)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	return allPartitions, nil
}

func CheckDiskSize() error {
	partitions, err := listPartitions()
	if err != nil {
		slog.Error("failed to fetch partitions", slog.String("error", err.Error()))
		return err
	}

	for _, p := range partitions {
		disk, err := gpud.Usage(p.Mountpoint)
		if err != nil {
			slog.Error("failed to fetch file system usage", slog.String("error", err.Error()))
			return err
		}

		if p.Mountpoint == "/boot" || p.Mountpoint == "/" {
			if disk.Free <= diskFreeSize[p.Mountpoint] {
				errMsg := fmt.Sprintf("%s has %v bytes of space left, exiting", p.Mountpoint, disk.Free)
				slog.Error(errMsg)
				return errors.New(errMsg)
			}
		}
	}

	return nil
}
