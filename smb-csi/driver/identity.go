package driver

import (
	"context"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/klog/v2"
)


func (d *Driver) GetPluginInfo(ctx context.Context, request *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	resp := &csi.GetPluginInfoResponse{
		Name:          driverName,
		VendorVersion: driverVersion,
	}

	klog.Infof("Plugin-info: %+v", resp)
	return resp, nil
}

func (d *Driver) GetPluginCapabilities(ctx context.Context, request *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	resp := &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
					},
				},
			},
		},
	}

	klog.Infof("Plugin-Capabilities: %+v", resp)
	return resp, nil
}

func (d *Driver) Probe(ctx context.Context, request *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	resp := &csi.ProbeResponse{
		Ready: wrapperspb.Bool(true),
	}

	klog.Infof("Probe: %+v", resp)
	return resp, nil

}
