package driver

import (
	"context"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	"smb-csi/driver/healtchCheck"
	"smb-csi/driver/state"
	"sort"
	"strconv"
)

func (d *Driver) CreateVolume(ctx context.Context, request *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {

	requestCapacity := request.GetCapacityRange().GetRequiredBytes()
	requestContentSource := request.GetVolumeContentSource()
	requestName := request.GetName()
	requestParameters := request.GetParameters()

	if requestName == "" { return nil, status.Error(codes.InvalidArgument, "No VolumeID specified") }
	if len(requestParameters) == 0  {return nil, status.Error(codes.InvalidArgument, "PV should contain at least the source attribute")}

	vol, err := d.State.GetVolumeByName(requestName)
	if err == nil {
		if vol.VolSize < requestCapacity {
			return nil, status.Error(codes.AlreadyExists, "Requested PV does already exist, but is to small for the request")
		} else {
			klog.Infof("Volume already exists")
			return &csi.CreateVolumeResponse{
				Volume: &csi.Volume{
					VolumeId:           vol.VolID,
					CapacityBytes:      vol.VolSize,
					VolumeContext:      requestParameters,
					ContentSource:      requestContentSource,
				},
			}, nil
		}
	}

	volumeID := string(uuid.NewUUID())
	path := filepath.Join(d.StateDir, volumeID)
	if mountErr := os.MkdirAll(path, 0777); mountErr != nil {
		klog.Infof("Failed create state dir %s", mountErr.Error())
		return nil, status.Error(codes.Internal, "Failed creating local PV")
	}
	klog.Info("Created state dir")

	newVol := state.Volume{
		VolID: volumeID,
		VolName: requestName,
		VolSize: requestCapacity,
		VolPath: filepath.Join(d.StateDir, volumeID),
		VolAccessType: state.MountAccess,
		Ephemeral: false,
	}

	if updateErr := d.State.UpdateVolume(newVol); updateErr != nil {
		return nil, status.Error(codes.Internal, updateErr.Error())
	}

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId: volumeID,
			VolumeContext: requestParameters,
			CapacityBytes: requestCapacity,
			ContentSource: requestContentSource,
		},
	}, nil
}

func (d *Driver) DeleteVolume(ctx context.Context, request *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {

	requestID := request.GetVolumeId()
	path := filepath.Join(d.StateDir, requestID)

	vol, err := d.State.GetVolumeByID(requestID)
	if err != nil {
		return &csi.DeleteVolumeResponse{}, nil
	}
	if vol.IsAttached || vol.IsPublished || vol. IsStaged {
		return nil, status.Error(codes.FailedPrecondition, "PV Cannot be deleted while in use")
	}

	if removeDirErr := os.RemoveAll(path); removeDirErr != nil && os.IsNotExist(removeDirErr) {
		return nil, status.Error(codes.Internal, removeDirErr.Error())
	}

	if deleteVolErr := d.State.DeleteVolume(requestID); deleteVolErr != nil {
		return nil, status.Error(codes.Internal, deleteVolErr.Error())
	}

	return &csi.DeleteVolumeResponse{}, nil
}

func (d *Driver) ControllerPublishVolume(ctx context.Context, request *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {

	vol, err := d.State.GetVolumeByID(request.GetVolumeId())
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed Getting Volume by ID")
	}

	nodeID := request.GetNodeId()
	if nodeID == "" {
		return nil, status.Error(codes.InvalidArgument, "NodeID missing")
	}

	vol.NodeID = nodeID
	vol.IsAttached = true

	if updateErr := d.State.UpdateVolume(vol); updateErr != nil {
		return nil, status.Error(codes.Internal, "Failed updating Volume State")
	}

	return &csi.ControllerPublishVolumeResponse{}, nil
	// return nil, status.Error(codes.Unavailable,"Filesystem-attach is not supported")
}

func (d *Driver) ControllerUnpublishVolume(ctx context.Context, request *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {

	vol, err := d.State.GetVolumeByID(request.GetVolumeId())
	if err != nil {
		return &csi.ControllerUnpublishVolumeResponse{}, nil
	}

	vol.NodeID = ""
	vol.IsAttached = false

	if updateErr := d.State.UpdateVolume(vol); updateErr != nil {
		return nil, status.Error(codes.Internal, "Failed updating Volume State")
	}

	return &csi.ControllerUnpublishVolumeResponse{}, nil
	// return nil, status.Error(codes.Unavailable,"Filesystem-detach is not supported")
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
		healthy, reason := healtchCheck.HealthCheck(vol)

		entries = append(entries, &csi.ListVolumesResponse_Entry{
			Volume: &csi.Volume{
				VolumeId: vol.VolID,
				CapacityBytes: vol.VolSize,
			},
			Status: &csi.ListVolumesResponse_VolumeStatus{
				PublishedNodeIds: []string{vol.NodeID},
				VolumeCondition: &csi.VolumeCondition{
					Abnormal: !healthy,
					Message: reason,
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
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
		csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
		csi.ControllerServiceCapability_RPC_VOLUME_CONDITION,
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
	panic("implement me")
}

func (d *Driver) DeleteSnapshot(ctx context.Context, request *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	panic("implement me")
}

func (d *Driver) ListSnapshots(ctx context.Context, request *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	panic("implement me")
}

func (d *Driver) ControllerExpandVolume(ctx context.Context, request *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	panic("implement me")
}

func (d *Driver) ControllerGetVolume(ctx context.Context, request *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	panic("implement me")
}

