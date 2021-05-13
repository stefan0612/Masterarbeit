package test

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetPluginInfo_NameAndVersion(t *testing.T) {
	req := csi.GetPluginInfoRequest{}
	resp, err := d.GetPluginInfo(ctx, &req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, d.Name, resp.Name)
	assert.Equal(t, d.Version, resp.VendorVersion)
}

func TestGetPluginCapabilities_CapsNotEmpty(t *testing.T) {
	req := csi.GetPluginCapabilitiesRequest{}
	resp, err := d.GetPluginCapabilities(ctx, &req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Capabilities)
}

func TestProbe(t *testing.T) {
	req := csi.ProbeRequest{}
	resp, err := d.Probe(ctx, &req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Ready.Value)
}
