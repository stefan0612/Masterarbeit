package driver

import (
	"context"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (d *Driver) CreateVolume(ctx context.Context, request *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {

	if request.GetName() == "" { return nil, status.Error(codes.InvalidArgument, "No VolumeID specified") }
	if len(request.GetParameters()) == 0  {return nil, status.Error(codes.InvalidArgument, "PV should contain at least the source attribute")}

	resp := &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId: request.GetName(),
			VolumeContext: request.GetParameters(),
			CapacityBytes: request.GetCapacityRange().GetRequiredBytes(),
		},
	}

	return resp, nil
}

func (d *Driver) DeleteVolume(ctx context.Context, request *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {

	resp := &csi.DeleteVolumeResponse{}

	return resp, nil
}

func (d *Driver) ControllerPublishVolume(ctx context.Context, request *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unavailable,"Filesystem-attach is not supported")
}

func (d *Driver) ControllerUnpublishVolume(ctx context.Context, request *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unavailable,"Filesystem-detach is not supported")
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

	if d.restClient == nil {
		return nil, status.Error(codes.Unavailable, "Not available because rest-client is missing")
	}

	pvClient := d.restClient.CoreV1().PersistentVolumes()
	pvList, err := pvClient.List(metav1.ListOptions{})

	if err != nil {
		return nil, status.Error(codes.Canceled, "Failed getting volume list")
	}

	var volumes []*csi.ListVolumesResponse_Entry
	for _, pv := range pvList.Items {

		if pv.Status.Phase != "Available" ||
			pv.Spec.PersistentVolumeSource.CSI.Driver != driverName { continue }

		volumes = append(volumes, &csi.ListVolumesResponse_Entry{
			Volume: &csi.Volume{
				VolumeId: pv.GetName(),
				VolumeContext: pv.Spec.PersistentVolumeSource.CSI.VolumeAttributes,
			},
			Status: &csi.ListVolumesResponse_VolumeStatus{
				VolumeCondition: &csi.VolumeCondition{
					Abnormal: pv.Status.Reason != "",
					Message: pv.Status.Message,
				},
			},
		})
	}

	return &csi.ListVolumesResponse{
		Entries: volumes,
	}, nil
}

func (d *Driver) GetCapacity(ctx context.Context, request *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	panic("implement me")
}

func (d *Driver) ControllerGetCapabilities(ctx context.Context, request *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {

	capabilities := []csi.ControllerServiceCapability_RPC_Type {
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
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
