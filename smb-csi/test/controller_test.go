package test

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"testing"
)

const testVolName = "testName4"

func TestCreateVolume_NoServer(t *testing.T) {
	req := csi.CreateVolumeRequest{
		Name: testVolName,
		Parameters: map[string]string{"share": "/share1"},
	}
	resp, err := d.CreateVolume(ctx, &req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestCreateVolume_NoShare(t *testing.T) {
	req := csi.CreateVolumeRequest{
		Name: testVolName,
		Parameters: map[string]string{"server": "127.0.0.1"},
	}
	resp, err := d.CreateVolume(ctx, &req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestPublishVolume_NoServer(t *testing.T) {
	req := csi.ControllerPublishVolumeRequest{
		NodeId: "test",
		VolumeId: "testID",
		VolumeContext: map[string]string{"share": "/share1"},
	}
	resp, err := d.ControllerPublishVolume(ctx, &req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestPublishVolume_NoShare(t *testing.T) {
	req := csi.ControllerPublishVolumeRequest{
		NodeId: "test",
		VolumeId: "testID",
		VolumeContext: map[string]string{"server": "127.0.0.1"},
	}
	resp, err := d.ControllerPublishVolume(ctx, &req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestDeleteVolume(t *testing.T) {
	req := csi.DeleteVolumeRequest{
		VolumeId: "testID",
	}
	resp, err := d.DeleteVolume(ctx, &req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
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


