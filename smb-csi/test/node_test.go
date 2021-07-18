package test

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"testing"
)

/*
	Difficult to test success, because method relies on external
	parameters like smb server and credentials
*/

func TestNodeStageVolume_NoArguments(t *testing.T) {
	req := csi.NodeStageVolumeRequest{}
	resp, err := d.NodeStageVolume(ctx, &req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestNodeUnstageVolume_NoArguments(t *testing.T) {
	req := csi.NodeUnstageVolumeRequest{}
	resp, err := d.NodeUnstageVolume(ctx, &req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestNodePublishVolume_NoArguments(t *testing.T) {
	req := csi.NodePublishVolumeRequest{}
	resp, err := d.NodePublishVolume(ctx, &req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestNodeUnpublishVolume_NoArguments(t *testing.T) {
	req := csi.NodeUnpublishVolumeRequest{}
	resp, err := d.NodeUnpublishVolume(ctx, &req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestNodeGetInfo(t *testing.T) {
	req := csi.NodeGetInfoRequest{}
	resp, err := d.NodeGetInfo(ctx, &req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, d.NodeID, resp.NodeId)
}

func TestNodeCapabilities_NotEmpty(t *testing.T) {
	req := csi.NodeGetCapabilitiesRequest{}
	resp, err := d.NodeGetCapabilities(ctx, &req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Capabilities)
}

func TestNodeGetVolumeStats_NoVolumePath(t *testing.T) {
	req := csi.NodeGetVolumeStatsRequest{}
	resp, err := d.NodeGetVolumeStats(ctx, &req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}
