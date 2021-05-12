package main

import (
	"flag"
	"k8s.io/klog/v2"
	"os"
	smb "smb-csi/driver"
	"strings"
)

var (
	endpoint = flag.String("endpoint","/csi/csi.sock","CSI UNIX Domain Socket Endpoint")
	nodeid = flag.String("nodeid","","ID of Node passed from kube args")
)

func main() {
	flag.Parse()
	if *endpoint == "" {
		klog.Fatalln("No valid UNIX Domain Socket specified")
	}
	if *nodeid == "" {
		klog.Fatalln("NodeID is missing")
	}
	if !strings.HasPrefix(*endpoint, "unix://") {
		*endpoint = "unix://" + *endpoint
	}
	driver, driverErr := smb.NewDriver(*nodeid)
	if driverErr != nil {
		klog.Fatalln(driverErr)
		os.Exit(1)
	}

	klog.Infof("Driver created on following Node: %s", *nodeid)

	err := driver.Run(*endpoint)
	if err != nil {
		klog.Fatalln(err)
	}
}
