package mounter

import (
	"golang.org/x/sys/unix"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	"os"
	"strings"
)

type Mounter interface {
	GetFilesystemInfo(path string) (*unix.Statfs_t, error)
	PathExists(path string) bool
	CreateDir(path string, mode os.FileMode) error
	DeleteDir(path string) error
	Mount(src string, target string, mountOptions []string) error
	BindMount(src string, target string) error
	Unmount(target string) error
}

type BaseMounter struct {}

func NewMounter() *Mounter {
	var baseMounter Mounter
	baseMounter = &BaseMounter{}
	return &baseMounter
}

func (*BaseMounter) GetFilesystemInfo(path string) (*unix.Statfs_t, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Error while getting volume stats")
	}
	return &stat, nil
}

func (*BaseMounter) PathExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

func (*BaseMounter) CreateDir(path string, mode os.FileMode) error {
	if createErr := os.MkdirAll(path, mode); createErr != nil {
		return createErr
	}
	return nil
}

func (*BaseMounter) DeleteDir(path string) error {
	if deleteDirErr := os.RemoveAll(path); deleteDirErr != nil {
		return deleteDirErr
	}
	return nil
}

func (*BaseMounter) Mount(src string, target string, mountOptions []string) error {

	//Check if the target path exist, else create it
	if createDirErr := os.MkdirAll(target, os.ModeDir); createDirErr != nil {
		return status.Errorf(codes.Internal, "Failed creating mount directory: %s", createDirErr.Error())
	}

	// Build string from mountOptions array, separated by ","
	mountOptionsString := strings.Join(mountOptions, ",")

	// Try to mount directory, else try to remove the created mounting-point directory and return error
	if mountErr := unix.Mount(src, target, "cifs", 0, mountOptionsString); mountErr != nil {
		return status.Error(codes.Internal,"Failed mounting directory")
	}
	return nil
}

func (*BaseMounter) BindMount(src string, target string) error {

	//Check if the source path exist
	if _, statSourceErr := os.Stat(src); statSourceErr != nil {
		return status.Error(codes.InvalidArgument,"Staging path does not exist")
	}

	//Check if the target path exist, else create it
	if createDirErr := os.MkdirAll(target, os.ModeDir); createDirErr != nil {
		return status.Error(codes.Internal,"Failed creating mount directory")
	}

	// Try to mount directory, else try to remove the created mounting-point directory and return error
	if mountErr := unix.Mount(src, target, "bind", unix.MS_BIND, ""); mountErr != nil {
		if deleteDirErr := os.RemoveAll(target); deleteDirErr == nil {
			return status.Error(codes.Internal,"Failed cleanup after mounting failed")
		}
		return status.Error(codes.Internal,"Failed mounting directory")
	}

	return nil
}

func (*BaseMounter) Unmount(target string) error {

	if _, statErr := os.Stat(target); statErr != nil { return nil }
	klog.Infof("Target Unmounting: %s", target)
	if unmountErr := unix.Unmount(target, 0); unmountErr != nil {
		return unmountErr
		//return status.Error(codes.Internal,"Failed unmounting directory")
	}

	return nil
}
