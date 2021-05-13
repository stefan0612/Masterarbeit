package healtchCheck

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"smb-csi/driver/state"
)

func HealthCheck(vol state.Volume) (bool, string) {

	spExist, err := checkPathExist(vol.VolPath)
	if err != nil {
		return false, err.Error()
	}
	if !spExist {
		return false, "The source path of the volume doesn't exist"
	}

	capAndUsageValid, err := checkPVCapacityAndUsageValid(vol)
	if err != nil {
		return false, err.Error()
	}

	if !capAndUsageValid {
		return false, "The capacity of volume is greater than actual storage"
	}

	return true, ""
}

func FsInfo(path string) (int64, int64, int64, int64, int64, int64, error) {
	statfs := &unix.Statfs_t{}
	err := unix.Statfs(path, statfs)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, err
	}

	// Available is blocks available * fragment size
	available := int64(statfs.Bavail) * statfs.Bsize

	// Capacity is total block count * fragment size
	capacity := int64(statfs.Blocks) * statfs.Bsize

	// Usage is block being used * fragment size (aka block size).
	usage := (int64(statfs.Blocks) - int64(statfs.Bfree)) * statfs.Bsize

	inodes := int64(statfs.Files)
	inodesFree := int64(statfs.Ffree)
	inodesUsed := inodes - inodesFree

	return available, capacity, usage, inodes, inodesFree, inodesUsed, nil
}

func checkPathExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}
func checkPVCapacityAndUsageValid(volume state.Volume) (bool, error) {

	volumePath := volume.VolPath

	fsavailable, fscapacity, _, _, _, _, err := FsInfo(volumePath)
	if err != nil {
		return false, fmt.Errorf("failed to get capacity info: %+v", err)
	}
	volumeCapacity := volume.VolSize
	if fscapacity < volumeCapacity {
		return false, fmt.Errorf("volume capacity surpassed filesystem capacity")
	}
	if fsavailable <= 0 {
		return false, fmt.Errorf("no more available space")
	}

	return true, nil
}
