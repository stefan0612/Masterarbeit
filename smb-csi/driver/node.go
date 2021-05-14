package driver

import (
	"context"
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	"strings"
)

func (d *Driver) NodeStageVolume(ctx context.Context, request *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {

	// Get all necessary mounting information from the request parameter
	targetPath := request.GetStagingTargetPath()
	volumeContext := request.GetVolumeContext()
	secrets := request.GetSecrets()
	mountFlags := request.GetVolumeCapability().GetMount().GetMountFlags()

	//Check if the target path exist, else create it
	if _, statErr := os.Stat(targetPath); statErr != nil {
		if createDirErr := os.Mkdir(targetPath, os.ModeDir); createDirErr != nil {
			return nil, status.Errorf(codes.Internal,"Failed creating mount directory: %s", createDirErr.Error())
		}
	}

	// Check if source path is present
	source, isSourcePresent := volumeContext["source"]
	if !isSourcePresent {
		return nil, status.Error(codes.InvalidArgument,"No smb-share source is present")
	}

	// Check if  username (optional) is present, else log that no username was provided
	username, isUsernamePresent := secrets["username"]
	if !isUsernamePresent {
		klog.Info("No username specified in secrets")
	}

	// Check if  password (optional) is present, else log that no password was provided
	password, isPasswordPresent := secrets["password"]
	if !isPasswordPresent {
		klog.Info("No password specified in secrets")
	}

	// Append all necessary mount-args to an array
	var mountOptions []string
	mountOptions = append(mountOptions, fmt.Sprintf("username=%s", username))
	mountOptions = append(mountOptions, fmt.Sprintf("password=%s", password))
	mountOptions = append(mountOptions, fmt.Sprintf("vers=%s", "3.0"))
	mountOptions = append(mountOptions, mountFlags...)

	// Build string from mountOptions array, separated by ","
	mountOptionsString := strings.Join(mountOptions, ",")

	// Try to mount directory, else try to remove the created mounting-point directory and return error
	if mountErr := unix.Mount(source, targetPath, "cifs", 0, mountOptionsString); mountErr != nil {
		klog.Errorf("Error while mounting: %s", mountErr)
		if deleteDirErr := os.RemoveAll(targetPath); deleteDirErr != nil {
			return nil, status.Error(codes.Internal,"Failed cleanup after mounting failed")
		}
		return nil, status.Error(codes.Internal,"Failed mounting directory")
	}

	return &csi.NodeStageVolumeResponse{}, nil
}

func (d *Driver) NodeUnstageVolume(ctx context.Context, request *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {

	targetPath := request.GetStagingTargetPath()
	if _, statErr := os.Stat(targetPath); statErr != nil {
		//return nil, status.Error(codes.InvalidArgument,"Specified target directory does not exist")
		return &csi.NodeUnstageVolumeResponse{}, nil
	}

	if unmountErr := unix.Unmount(targetPath, 0); unmountErr != nil {
		return nil, status.Error(codes.Internal,"Failed unmounting directory")
	}

	if deleteDirErr := os.RemoveAll(targetPath); deleteDirErr != nil {
		return nil, status.Error(codes.Internal,"Failed removing directory after unmount")
	}

	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (d *Driver) NodePublishVolume(ctx context.Context, request *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {

	stagingPath := request.GetStagingTargetPath()
	targetPath := request.GetTargetPath()

	//Check if the source path exist
	if _, statSourceErr := os.Stat(stagingPath); statSourceErr != nil {
		return nil, status.Error(codes.InvalidArgument,"Staging path does not exist")
	}

	//Check if the target path exist, else create it
	if _, statTargetErr := os.Stat(targetPath); statTargetErr != nil {
		if createDirErr := os.Mkdir(targetPath, os.ModeDir); createDirErr != nil {
			return nil, status.Error(codes.Internal,"Failed creating mount directory")
		}
	}

	// Try to mount directory, else try to remove the created mounting-point directory and return error
	if mountErr := unix.Mount(stagingPath, targetPath, "bind", unix.MS_BIND, ""); mountErr != nil {
		klog.Errorf("Error while mounting: %s", mountErr)
		if deleteDirErr := os.RemoveAll(targetPath); deleteDirErr == nil {
			return nil, status.Error(codes.Internal,"Failed cleanup after mounting failed")
		}
		return nil, status.Error(codes.Internal,"Failed mounting directory")
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (d *Driver) NodeUnpublishVolume(ctx context.Context, request *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {

	targetPath := request.GetTargetPath()
	if _, statErr := os.Stat(targetPath); statErr != nil {
		return &csi.NodeUnpublishVolumeResponse{}, nil
		//return nil, status.Error(codes.InvalidArgument,"Specified target directory does not exist")
	}

	if unmountErr := unix.Unmount(targetPath, 0); unmountErr != nil {
		return nil, status.Error(codes.Internal,"Failed unmounting directory")
	}

	if deleteDirErr := os.RemoveAll(targetPath); deleteDirErr != nil {
		return nil, status.Error(codes.Internal,"Failed removing directory after unmount")
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (d *Driver) NodeGetVolumeStats(ctx context.Context, request *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {

	volumePath := request.GetVolumePath()
	if volumePath == "" {
		return nil, status.Error(codes.InvalidArgument, "Volume Path is missing")
	}

	var stat unix.Statfs_t
	if err := unix.Statfs(volumePath, &stat); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Error while getting volume stats")
	}

	var totalSize int64
	var used int64
	var available int64

	// Total Disk Size on Current Node
	totalSize = stat.Bsize * int64(stat.Blocks)

	// Total Used Space from given Node, calculated by iterating over every file and adding the filesize to a counter
	fileTraverseErr := filepath.Walk(volumePath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			used += info.Size()
		}
		return err
	})
	if fileTraverseErr != nil {
		return nil, status.Error(codes.Internal, "Failed calculating used space")
	}

	// Available Size, calculated by subtracting used space from total space
	available = totalSize - used

	return &csi.NodeGetVolumeStatsResponse{
		Usage: []*csi.VolumeUsage{
			{
				Unit: csi.VolumeUsage_BYTES,
				Total: totalSize,
				Available: available,
				Used: used,
			},
		},
	}, nil
}

func (d *Driver) NodeExpandVolume(ctx context.Context, request *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Volume expansion is not supported for Filesystems")
}

func (d *Driver) NodeGetCapabilities(ctx context.Context, request *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {

	capabilities := []csi.NodeServiceCapability_RPC_Type {
		csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
		csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
	}

	var capabilityObjects []*csi.NodeServiceCapability
	for _, capability := range capabilities {
		capabilityObjects = append(capabilityObjects, &csi.NodeServiceCapability{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: capability,
				},
			},
		})
	}

	resp := &csi.NodeGetCapabilitiesResponse{
		Capabilities: capabilityObjects,
	}

	return resp, nil
}

func (d *Driver) NodeGetInfo(ctx context.Context, request *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {

	resp := &csi.NodeGetInfoResponse{
		NodeId: d.NodeID,
	}

	return resp, nil
}
