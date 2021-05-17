package mock

import (
	"golang.org/x/sys/unix"
	"os"
	"smb-csi/driver/mounter"
)

type FakeMounter struct {}

func NewFakeMounter() *mounter.Mounter {
	var fakeMounter mounter.Mounter
	fakeMounter = &FakeMounter{}
	return &fakeMounter
}

func (*FakeMounter) GetFilesystemInfo(path string) (*unix.Statfs_t, error) {
	stat := unix.Statfs_t {
		Bsize: 1000,
		Blocks: 8,
	}
	return &stat, nil
}

func (*FakeMounter) PathExists(path string) bool {
	return true
}

func (*FakeMounter) CreateDir(path string, mode os.FileMode) error {
	return nil
}

func (*FakeMounter) DeleteDir(path string) error {
	return nil
}

func (*FakeMounter) Mount(src string, target string, mountOptions []string) error {
	return nil
}

func (*FakeMounter) BindMount(src string, target string) error {
	return nil
}

func (*FakeMounter) Unmount(target string) error {
	return nil
}
