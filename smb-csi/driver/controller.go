package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/otiai10/copy"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	"smb-csi/driver/snapshotter"
	"sort"
	"strconv"
	"strings"
)

func (d *Driver) CreateVolume(ctx context.Context, request *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	klog.Infof("create request: %+v", request)

	requestCapacity := request.GetCapacityRange().GetRequiredBytes()
	requestContentSource := request.GetVolumeContentSource()
	requestedVolumeID := request.GetName()
	requestParameters := request.GetParameters()
	share, isSharePresent := requestParameters["share"]
	if !isSharePresent {
		return nil, status.Error(codes.InvalidArgument,"No smb-share source is present")
	}
	server, isServerPresent := requestParameters["server"]
	if !isServerPresent {
		return nil, status.Error(codes.InvalidArgument,"No smb-server source is present")
	}

	if requestedVolumeID == "" { return nil, status.Error(codes.InvalidArgument, "No VolumeID specified") }

	if vol, err := d.PVClient.Get(ctx, requestedVolumeID, v1.GetOptions{}); err == nil {
		return &csi.CreateVolumeResponse{
			Volume: &csi.Volume{
				VolumeId: vol.GetName(),
				VolumeContext: requestParameters,
				ContentSource: requestContentSource,
			},
		}, nil
	}

	resp := &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId: requestedVolumeID,
			VolumeContext: requestParameters,
			CapacityBytes: requestCapacity,
		},
	}

	serverSharePath := "//" + strings.Join([]string{server, share}, "/")
	localSharePath := filepath.Join(driverStateDir, share)
	localVolumePath := filepath.Join(localSharePath, requestedVolumeID)
	if err := d.Mounter.AuthMount(serverSharePath, localSharePath, request.GetSecrets(), nil); err != nil {
		klog.Infof("Failed: %s", err.Error())
		return nil, err
	}
	defer d.Mounter.Unmount(localSharePath)

	if err := d.Mounter.CreateDir(localVolumePath, os.ModeDir); err != nil {
		klog.Infof("Failed: %s", err.Error())
		return nil, err
	}

	switch requestContentSource.GetType().(type) {
	case *csi.VolumeContentSource_Snapshot:
		snapID := requestContentSource.GetSnapshot().GetSnapshotId()
		snap, err := d.RestClient.Resource(schema.GroupVersionResource{
			Group: "snapshot.storage.k8s.io",
			Resource: "volumesnapshotcontents",
			Version: "v1",
		}).Get(ctx, strings.Replace(snapID, "shot", "content", 1), v1.GetOptions{})
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Requested Snap with ID: %s does not exist", snapID)
		}
		spec := snap.Object["spec"].(map[string]interface{})
		rollbackVolID := spec["source"].(map[string]interface{})["volumeHandle"].(string)

		rollbackPV, err := d.PVClient.Get(ctx, rollbackVolID, v1.GetOptions{})
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Volume referenced by snapshot does not exist", rollbackVolID)
		}
		res, _ :=json.MarshalIndent(rollbackPV,"", "\t")
		klog.Info(string(res))

		rollbackServer := rollbackPV.Spec.CSI.VolumeAttributes["server"]
		rollbackShare := rollbackPV.Spec.CSI.VolumeAttributes["share"]

		rollbackServerVolumePath := "//" + strings.Join([]string{rollbackServer, rollbackShare, rollbackVolID}, "/")
		rollbackLocalSharePath := filepath.Join(driverStateDir, share)
		rollbackLocalVolumePath := filepath.Join(rollbackLocalSharePath, rollbackVolID)


		if rollbackShare != share {
			if err := d.Mounter.AuthMount(rollbackServerVolumePath, rollbackLocalVolumePath, request.GetSecrets(), nil); err != nil {
				klog.Infof("Failed: %s", err.Error())
				return nil, err
			}
		}

		snapFile := fmt.Sprintf("%s/%s.snap", rollbackLocalVolumePath, snapID)
		if err := snapshotter.ExtractSnap(snapFile, localVolumePath); err != nil {
			klog.Infof("Failed: %s", err.Error())
			return nil, err
		}

		resp.Volume.ContentSource = &csi.VolumeContentSource{
			Type: &csi.VolumeContentSource_Snapshot{
				Snapshot: &csi.VolumeContentSource_SnapshotSource{
					SnapshotId: snapID,
				},
			},
		}
	case *csi.VolumeContentSource_Volume:
		rollbackVolID := requestContentSource.GetVolume().GetVolumeId()
		rollbackPV, err := d.PVClient.Get(ctx, rollbackVolID, v1.GetOptions{})
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Requested Volume with ID: %s does not exist", rollbackVolID)
		}
		rollbackServer := rollbackPV.Spec.CSI.VolumeAttributes["server"]
		rollbackShare := rollbackPV.Spec.CSI.VolumeAttributes["share"]

		rollbackServerVolumePath := "//" + strings.Join([]string{rollbackServer, rollbackShare, rollbackVolID}, "/")
		rollbackLocalVolumePath := filepath.Join(driverStateDir, share, rollbackVolID)

		if rollbackShare != share {
			if err := d.Mounter.AuthMount(rollbackServerVolumePath, rollbackLocalVolumePath, request.GetSecrets(), nil); err != nil {
				klog.Infof("Failed: %s", err.Error())
				return nil, err
			}
		}

		if err := copy.Copy(rollbackLocalVolumePath, localVolumePath); err != nil {
			klog.Infof("Failed populating Volume with other Volumestate: %s", err.Error())
			break
		}

		if err := d.Mounter.Unmount(rollbackLocalVolumePath); err != nil {
			klog.Infof("failed unmounting local rollback vol path: %s", err.Error())
		}

		resp.Volume.ContentSource = &csi.VolumeContentSource{
			Type: &csi.VolumeContentSource_Volume{
				Volume: &csi.VolumeContentSource_VolumeSource{
					VolumeId: requestContentSource.GetVolume().GetVolumeId(),
				},
			},
		}
	default:
		break
	}

	return resp, nil
}

func (d *Driver) DeleteVolume(ctx context.Context, request *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	return &csi.DeleteVolumeResponse{}, nil
}

func (d *Driver) ControllerPublishVolume(ctx context.Context, request *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {

	volumeId := request.GetVolumeId()
	volumeContext := request.GetVolumeContext()
	secrets := request.GetSecrets()
	mountFlags := request.GetVolumeCapability().GetMount().GetMountFlags()

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

	if err := d.Mounter.AuthMount(sourceMountPoint, filepath.Join(d.StateDir, share), secrets, mountFlags); err != nil {
		klog.Infof("Failed: %s", err.Error())
		return nil, err
	}

	if err := d.Mounter.CreateDir(filepath.Join(d.StateDir, share, volumeId), os.ModeDir); err != nil {
		return nil, err
	}

	_ = d.Mounter.Unmount(filepath.Join(d.StateDir, share))

	return &csi.ControllerPublishVolumeResponse{}, nil
}

func (d *Driver) ControllerUnpublishVolume(ctx context.Context, request *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {

	volumeID := request.GetVolumeId()

	pv, err := d.PVClient.Get(ctx, volumeID, v1.GetOptions{})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument,"Cannot find Volume with ID: %s", volumeID)
	}

	share := pv.Spec.CSI.VolumeAttributes["share"]

	_ = d.Mounter.Unmount(filepath.Join(d.StateDir, share, volumeID))

	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

func (d *Driver) ValidateVolumeCapabilities(ctx context.Context, request *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {

	// These are all supported Volume Capability Modes, which SMB supports
	supportedModes := []csi.VolumeCapability_AccessMode_Mode{
		csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
		csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY,
		csi.VolumeCapability_AccessMode_MULTI_NODE_SINGLE_WRITER,
		csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY,
		csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
	}

	var supportedCaps []*csi.VolumeCapability
	for _, mode := range supportedModes {
		supportedCaps = append(supportedCaps, &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{
				Mount: &csi.VolumeCapability_MountVolume{
					FsType: "",
				},
			},
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: mode,
			},
		})
	}

	contains := func(slice []csi.VolumeCapability_AccessMode_Mode, val csi.VolumeCapability_AccessMode_Mode) bool {
		isInSlice := false
		for _, accessMode := range slice {
			if accessMode == val { isInSlice = true }
		}
		return isInSlice
	}

	// Check if all requested Capabilities are supported
	requestedCapabilities := request.GetVolumeCapabilities()
	for _, capability := range requestedCapabilities {
		// If at least one requested capability is not supported, set the error-message field of the response
		if !contains(supportedModes, capability.GetAccessMode().GetMode()) {
			return &csi.ValidateVolumeCapabilitiesResponse{
				Message: "Requested Capabilities not supported",
			}, nil
		}
	}

	// Return the supportedCaps and leave the error-message field empty
	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeCapabilities: supportedCaps,
		},
	}, nil
}

func (d *Driver) ListVolumes(ctx context.Context, request *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {

	var entries []*csi.ListVolumesResponse_Entry
	nextToken := ""

	vols := d.State.GetVolumes()
	volumeLength := len(vols)
	sort.Slice(vols, func(i, j int) bool {
		return vols[i].VolID < vols[j].VolID
	})

	maxEntries := int(request.GetMaxEntries())
	if maxEntries == 0 { maxEntries = volumeLength }
	if maxEntries < 0 { return nil, status.Error(codes.InvalidArgument, "Max Entries must not be negative or zero") }

	startingToken := request.GetStartingToken()
	if startingToken == "" { startingToken = "1" }

	startingIndex, parseErr := strconv.Atoi(startingToken)
	if parseErr != nil {
		return nil, status.Error(codes.InvalidArgument, "StartingToken must be a number")
	}
	if startingIndex > volumeLength {
		return &csi.ListVolumesResponse{}, nil
	}
	if startingIndex < 1 {
		startingIndex = 1
	}

	for index := startingIndex - 1; index < volumeLength && index < startingIndex + maxEntries - 1; index++ {

		vol := vols[index]
		//healthy, reason := healtchCheck.HealthCheck(vol)

		entries = append(entries, &csi.ListVolumesResponse_Entry{
			Volume: &csi.Volume{
				VolumeId: vol.VolID,
				CapacityBytes: vol.VolSize,
			},
			Status: &csi.ListVolumesResponse_VolumeStatus{
				PublishedNodeIds: []string{vol.NodeID},
				VolumeCondition: &csi.VolumeCondition{
					Abnormal: false,
					Message: "reason",
				},
			},
		})
	}

	if startingIndex + maxEntries <= volumeLength { nextToken = strconv.Itoa(startingIndex + maxEntries)}

	return &csi.ListVolumesResponse{
		Entries: entries,
		NextToken: nextToken,
	}, nil
}

func (d *Driver) GetCapacity(ctx context.Context, request *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	panic("implement me")
}

func (d *Driver) ControllerGetCapabilities(ctx context.Context, request *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {

	capabilities := []csi.ControllerServiceCapability_RPC_Type {
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT,
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS,
		csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
		csi.ControllerServiceCapability_RPC_VOLUME_CONDITION,
		csi.ControllerServiceCapability_RPC_CLONE_VOLUME,
	}

	var capabilityObjects []*csi.ControllerServiceCapability
	for _, capability := range capabilities {
		capabilityObjects = append(capabilityObjects, &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: capability,
				},
			},
		})
	}

	resp := &csi.ControllerGetCapabilitiesResponse{
		Capabilities: capabilityObjects,
	}

	return resp, nil
}

func (d *Driver) CreateSnapshot(ctx context.Context, request *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	requestVolID := request.GetSourceVolumeId()
	//requestName := strings.Replace(request.GetName(), "shot", "content", 1)
	requestName := request.GetName()
	secrets := request.GetSecrets()
	klog.Infof("VolID: %s", requestVolID)
	klog.Infof("SnapName: %s", requestName)
	klog.Infof("Create Snap request: %+v", request)

	// Return existing Snapshot if one exists
	existSnap, err := d.RestClient.Resource(schema.GroupVersionResource{
		Group: "snapshot.storage.k8s.io",
		Resource: "volumesnapshotcontents",
		Version: "v1",
	}).Get(ctx, requestName, v1.GetOptions{})
	if err == nil {

		snapSpec := existSnap.Object["spec"].(map[string]interface{})
		snapStatus := existSnap.Object["status"].(map[string]interface{})
		volID := snapSpec["source"].(map[string]interface{})["volumeHandle"].(string)
		snapID := snapStatus["snapshotHandle"].(string)
		restoreSize := snapStatus["restoreSize"].(int64)
		readyToUse := snapStatus["readyToUse"].(bool)


		return &csi.CreateSnapshotResponse{
			Snapshot: &csi.Snapshot{
				SnapshotId: snapID,
				SourceVolumeId: volID,
				// CreationTime: existingSnap.CreationTime,
				SizeBytes: restoreSize,
				ReadyToUse: readyToUse,
			},
		}, nil
	}

	pv, err := d.PVClient.Get(ctx, requestVolID, v1.GetOptions{})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Cannot find requested PV with id: %s", requestVolID)
	}
	volShare := pv.Spec.CSI.VolumeAttributes["share"]
	volServer := pv.Spec.CSI.VolumeAttributes["server"]
	volID := pv.GetName()
	volPath := filepath.Join(driverStateDir, volShare, volID)

	sourceVolumePoint := "//" + strings.Join([]string{volServer, volShare, volID}, "/")

	if err := d.Mounter.AuthMount(sourceVolumePoint, volPath, secrets, nil); err != nil {
		klog.Infof("Failed: %s", err.Error())
		return nil, err
	}

	snapshotID := requestName
	createdTime := timestamppb.Now()
	snapFile := filepath.Join(volPath, snapshotID) + ".snap"
	if err := snapshotter.CreateSnapshot(volPath, snapFile); err != nil {
		return nil, err
	}
	if err := d.Mounter.Unmount(volPath); err != nil {
		klog.Infof("Failed unmounting: %s", err.Error())
	}

	return &csi.CreateSnapshotResponse{
		Snapshot: &csi.Snapshot{
			SnapshotId: snapshotID,
			SourceVolumeId: volID,
			CreationTime: createdTime,
			// SizeBytes: snapshot.SizeBytes,
			ReadyToUse: true,
		},
	}, nil
}

func (d *Driver) DeleteSnapshot(ctx context.Context, request *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {

	requestSnapID := request.GetSnapshotId()
	secrets := request.GetSecrets()

	snap, snapErr := d.RestClient.Resource(schema.GroupVersionResource{
		Group: "snapshot.storage.k8s.io",
		Resource: "volumesnapshotcontents",
		Version: "v1",
	}).Get(ctx, strings.Replace(requestSnapID, "shot", "content", 1), v1.GetOptions{})
	if snapErr != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Cannot find Snapshot with ID: %s", requestSnapID)
	}

	spec := snap.Object["spec"].(map[string]interface{})
	volID := spec["source"].(map[string]interface{})["volumeHandle"].(string)
	volume, err := d.PVClient.Get(ctx, volID, v1.GetOptions{})
	if err != nil {
		// Referenced volume not found or already deleted, nothing to do
		return &csi.DeleteSnapshotResponse{}, nil
	}

	requestVolID := volume.GetName()
	share := volume.Spec.CSI.VolumeAttributes["share"]
	server := volume.Spec.CSI.VolumeAttributes["server"]
	path := filepath.Join(driverStateDir, share, volID)
	sourceVolumePoint := "//" + strings.Join([]string{server, share, requestVolID}, "/")

	if err := d.Mounter.AuthMount(sourceVolumePoint, path, secrets, nil); err != nil {
		klog.Infof("Failed: %s", err.Error())
		return nil, err
	}

	if err := snapshotter.DeleteSnapshot(filepath.Join(path, requestSnapID) + ".snap"); err != nil { return &csi.DeleteSnapshotResponse{}, nil }

	if err := d.Mounter.Unmount(path); err != nil {
		klog.Infof("Failed unmounting: %s", err.Error())
	}

	return &csi.DeleteSnapshotResponse{}, nil
}

func (d *Driver) ListSnapshots(ctx context.Context, request *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {

	var entries []*csi.ListSnapshotsResponse_Entry
	nextToken := ""

	snapshots := d.State.GetSnapshots()
	snapLength := len(snapshots)
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Id < snapshots[j].Id
	})

	maxEntries := int(request.GetMaxEntries())
	if maxEntries == 0 { maxEntries = snapLength }
	if maxEntries < 0 { return nil, status.Error(codes.InvalidArgument, "Max Entries must not be negative or zero") }

	startingToken := request.GetStartingToken()
	if startingToken == "" { startingToken = "1" }

	startingIndex, parseErr := strconv.Atoi(startingToken)
	if parseErr != nil {
		return nil, status.Error(codes.InvalidArgument, "StartingToken must be a number")
	}
	if startingIndex > snapLength {
		return &csi.ListSnapshotsResponse{}, nil
	}
	if startingIndex < 1 {
		startingIndex = 1
	}

	for index := startingIndex - 1; index < snapLength && index < startingIndex + maxEntries - 1; index++ {

		snap := snapshots[index]

		entries = append(entries, &csi.ListSnapshotsResponse_Entry{
			Snapshot: &csi.Snapshot{
				SnapshotId: snap.Id,
				SizeBytes: snap.SizeBytes,
				SourceVolumeId: snap.VolID,
				CreationTime: snap.CreationTime,
				ReadyToUse: snap.ReadyToUse,
			},
		})
	}

	if startingIndex + maxEntries <= snapLength { nextToken = strconv.Itoa(startingIndex + maxEntries)}

	return &csi.ListSnapshotsResponse{
		Entries: entries,
		NextToken: nextToken,
	}, nil
}

func (d *Driver) ControllerExpandVolume(ctx context.Context, request *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	panic("implement me")
}

func (d *Driver) ControllerGetVolume(ctx context.Context, request *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	panic("implement me")
}

