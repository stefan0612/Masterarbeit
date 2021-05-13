package test

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"smb-csi/driver/mock"
	"testing"
)
import "context"

var ctx = context.Background()
var d, _ = mock.NewMockDriver("test")
const testVolName = "testName4"

func TestCreateVolume(t *testing.T) {
	req := csi.CreateVolumeRequest{
		Name: testVolName,
		Parameters: map[string]string{"source": "/test"},
	}
	resp, err := d.CreateVolume(ctx, &req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestPublishVolume(t *testing.T) {
	vol, _ := d.State.GetVolumeByName(testVolName)
	req := csi.ControllerPublishVolumeRequest{
		NodeId: "test",
		VolumeId: vol.VolID,
	}
	resp, err := d.ControllerPublishVolume(ctx, &req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestUnpublishVolume(t *testing.T) {
	vol, _ := d.State.GetVolumeByName(testVolName)
	req := csi.ControllerUnpublishVolumeRequest{
		VolumeId: vol.VolID,
	}
	resp, err := d.ControllerUnpublishVolume(ctx, &req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestDeleteVolume(t *testing.T) {
	vol, _ := d.State.GetVolumeByName(testVolName)
	req := csi.DeleteVolumeRequest{
		VolumeId: vol.VolID,
	}
	resp, err := d.DeleteVolume(ctx, &req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestListVolumes_All(t *testing.T) {
	req := csi.ListVolumesRequest{}
	resp, err := d.ListVolumes(ctx, &req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, len(resp.Entries), len(d.State.GetVolumes()))
}

func TestListVolumes_MaxEntries(t *testing.T) {
	req := csi.ListVolumesRequest{
		MaxEntries: 2,
	}
	resp, err := d.ListVolumes(ctx, &req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, len(resp.Entries), 2)
}

func TestListVolumes_NegativeMaxEntries(t *testing.T) {
	req := csi.ListVolumesRequest{
		MaxEntries: -1,
	}
	resp, err := d.ListVolumes(ctx, &req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestListVolumes_VeryHighMaxEntries(t *testing.T) {
	req := csi.ListVolumesRequest{
		MaxEntries: 999,
	}
	resp, err := d.ListVolumes(ctx, &req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, len(resp.Entries), len(d.State.GetVolumes()))
}

func TestListVolumes_Pagination(t *testing.T) {
	req := csi.ListVolumesRequest{
		MaxEntries: 2,
	}
	resp, err := d.ListVolumes(ctx, &req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.NextToken)
	assert.Equal(t, len(resp.Entries), 2)

	req2:= csi.ListVolumesRequest{
		StartingToken: resp.NextToken,
	}
	resp2, err2 := d.ListVolumes(ctx, &req2)
	assert.NoError(t, err2)
	assert.NotNil(t, resp2)
	assert.Equal(t, len(resp2.Entries), len(d.State.GetVolumes()) - 2)
}

func TestValidateVolumeCapabilities(t *testing.T) {
	req := csi.ValidateVolumeCapabilitiesRequest{
		VolumeCapabilities: []*csi.VolumeCapability{
			{
				AccessMode: &csi.VolumeCapability_AccessMode{
					Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				},
			},
		},
	}
	resp, err := d.ValidateVolumeCapabilities(ctx, &req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Empty(t, resp.Message)
}


