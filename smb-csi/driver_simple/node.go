package driver

import (
	"context"
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	"os"
	"strings"
)

func (d *Driver) NodeStageVolume(ctx context.Context, request *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {

	// Get all necessary mounting information from the request parameter
	targetPath := request.GetStagingTargetPath()
	volumeContext := request.GetVolumeContext()
	secrets := request.GetSecrets()
	mountFlags := request.GetVolumeCapability().GetMount().GetMountFlags()
	volumeId := request.GetVolumeId()

	// Check if source path is present
	server, isServerPresent := volumeContext["server"]
	if !isServerPresent {
		return nil, status.Error(codes.InvalidArgument,"No smb-server source is present")
	}

	share, isSharePresent := volumeContext["share"]
	if !isSharePresent {
		return nil, status.Error(codes.InvalidArgument,"No smb-share source is present")
	}

	sourceMountPoint := "//" + strings.Join([]string{server, share}, "/")

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

	if err := d.Mounter.Mount(sourceMountPoint, targetPath, mountOptions); err != nil {
		klog.Infof("Failed: %s", err.Error())
		return nil, err
	}

	_ = d.Mounter.CreateDir(targetPath + "/" + volumeId, os.ModeDir)

	return &csi.NodeStageVolumeResponse{}, nil
}

func (d *Driver) NodeUnstageVolume(ctx context.Context, request *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {

	targetPath := request.GetStagingTargetPath()
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Target Path")
	}

	// Delete Directory on remote server when unmounting?

	if err := d.Mounter.Unmount(targetPath); err != nil {
		return nil, err
	}

	_ = d.Mounter.DeleteDir(targetPath)

	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (d *Driver) NodePublishVolume(ctx context.Context, request *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {

	stagingPath := request.GetStagingTargetPath()
	targetPath := request.GetTargetPath()
	volumeID := request.GetVolumeId()

	if err := d.Mounter.BindMount(stagingPath + "/" + volumeID, targetPath); err != nil {
		return nil, err
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (d *Driver) NodeUnpublishVolume(ctx context.Context, request *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {

	targetPath := request.GetTargetPath()

	if err := d.Mounter.Unmount(targetPath); err != nil {
		return nil, err
	}

	_ = d.Mounter.DeleteDir(targetPath)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (d *Driver) NodeGetVolumeStats(ctx context.Context, request *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	panic("implement me!")
}

func (d *Driver) NodeExpandVolume(ctx context.Context, request *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Volume expansion is not supported for Filesystems")
}

func (d *Driver) NodeGetCapabilities(ctx context.Context, request *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {

	capabilities := []csi.NodeServiceCapability_RPC_Type {
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
