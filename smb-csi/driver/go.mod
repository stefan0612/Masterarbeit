module smb-csi/driver

go 1.13

replace smb-csi/driver/state => ./state

require (
	github.com/container-storage-interface/spec v1.4.0
	golang.org/x/sys v0.0.0-20190826190057-c7b8b68b1456
	google.golang.org/grpc v1.37.1
	google.golang.org/protobuf v1.26.0
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	k8s.io/klog/v2 v2.8.0
	smb-csi/driver/state v1.0.0
)
