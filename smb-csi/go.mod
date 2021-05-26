module smb-csi

go 1.13

replace smb-csi/driver => ./driver

replace smb-csi/driver/state => ./driver/state

require (
	github.com/container-storage-interface/spec v1.4.0
	github.com/stretchr/testify v1.6.1
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.8.0
	smb-csi/driver v0.0.0-00010101000000-000000000000
)
