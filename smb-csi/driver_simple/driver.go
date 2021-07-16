package driver

import (
	"context"
	"fmt"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"smb-csi/driver/mounter"
)

const (
	driverName    = "seitenbau.csi.smb"
	driverVersion = "1.0.0"
)

type Driver struct {
	Name       string
	Version    string
	NodeID     string
	Mounter	   mounter.Mounter
	server     *grpc.Server
}

func NewDriver(nodeID string) (*Driver, error) {

	driver := &Driver{
		Name:     driverName,
		Version:  driverVersion,
		Mounter:  *mounter.NewMounter(),
		NodeID:   nodeID,
	}

	return driver, nil
}

func (d *Driver) Run(endpoint string) error {

	parsedUrl, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("can't parse endpoint: %q", err)
	}

	address := path.Join(parsedUrl.Host, filepath.FromSlash(parsedUrl.Path))
	if parsedUrl.Host == "" {
		address = filepath.FromSlash(parsedUrl.Path)
	}
	if err := os.Remove(address); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove unix domain socket file %s, error: %s", address, err)
	}

	listener, err := net.Listen(parsedUrl.Scheme, address)
	if err != nil {
		return fmt.Errorf("can't listen to: %v", err)
	}

	errHandler := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			klog.Errorf("method: %s, error: %s", info.FullMethod, err)
		}
		return resp, err
	}

	d.server = grpc.NewServer(grpc.UnaryInterceptor(errHandler))
	csi.RegisterIdentityServer(d.server, d)
	csi.RegisterControllerServer(d.server, d)
	csi.RegisterNodeServer(d.server, d)

	klog.Infof("Server Started! Listening to %s", address)

	return d.server.Serve(listener)
}

func (d *Driver) Stop() {
	klog.Info("Server Stopped!")
	d.server.Stop()
}
