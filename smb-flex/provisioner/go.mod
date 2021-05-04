module smb-flex/provisioner

go 1.13

replace (
	github.com/volume => ./volume
	k8s.io/api => k8s.io/api v0.17.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.2
	k8s.io/client-go => k8s.io/client-go v0.17.2
)

require (
	github.com/volume v0.0.0-00010101000000-000000000000
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/klog v1.0.0
	sigs.k8s.io/sig-storage-lib-external-provisioner v4.1.0+incompatible
)
