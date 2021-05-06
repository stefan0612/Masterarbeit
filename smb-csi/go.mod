module smb-csi

go 1.13

replace smb-csi/driver => ./driver

require (
	k8s.io/klog/v2 v2.8.0
	smb-csi/driver v0.0.0-00010101000000-000000000000
)
