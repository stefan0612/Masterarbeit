package driver

import (
	"context"
	"fmt"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"smb-csi/driver/mounter"
	"smb-csi/driver/state"
)

const (
	driverName    = "seitenbau.csi.smb"
	driverVersion = "1.0.0"
	driverStateDir	  = "/csi-data-dir"
)

type Driver struct {
	Name       string
	Version    string
	NodeID     string
	StateDir   string
	Mounter	   mounter.Mounter
	PVClient   v1.PersistentVolumeInterface
	RestClient dynamic.Interface
	server     *grpc.Server
	State      state.State
}

func NewDriver(nodeID string) (*Driver, error) {

	if stateDirErr := os.MkdirAll(driverStateDir, 0750); stateDirErr != nil {
		klog.Infof("Error creating state directory: %s", stateDirErr)
		return nil, stateDirErr
	}

	smbState := state.New(path.Join(driverStateDir, "state.json"))

	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Infof("Error creating cluster config: %s", err)
	}

	driver := &Driver{
		Name:     driverName,
		Version:  driverVersion,
		StateDir: driverStateDir,
		Mounter:  *mounter.NewMounter(),
		NodeID:   nodeID,
		State:    smbState,
	}

	client, err := kubernetes.NewForConfig(config)
	pvClient := client.CoreV1().PersistentVolumes()
	driver.PVClient = pvClient

	// No other possibility known to request existing volumesnapshots, because go-client has no snapshot support
	// Will not work with Version v1beta1 or v1alpha1, thus only use API-version v1 to create snapshots
	restClient, _ := dynamic.NewForConfig(config)
	driver.RestClient = restClient

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
