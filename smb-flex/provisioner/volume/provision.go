/*
Copyright 2016 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package volume

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/exec"
	"sigs.k8s.io/sig-storage-lib-external-provisioner/controller"
)

const (
	// are we allowed to set this? else make up our own
	annCreatedBy = "kubernetes.io/createdby"
	createdBy    = "flex-dynamic-provisioner"

	// A PV annotation for the identity of the flexProvisioner that provisioned it
	annProvisionerID = "Provisioner_Id"
)

// NewFlexProvisioner creates a new flex provisioner
func NewFlexProvisioner(client kubernetes.Interface, execCommand string, flexDriver string) controller.Provisioner {
	return newFlexProvisionerInternal(client, execCommand, flexDriver)
}

func newFlexProvisionerInternal(client kubernetes.Interface, execCommand string, flexDriver string) *flexProvisioner {
	var identity types.UID

	provisioner := &flexProvisioner{
		client:      client,
		execCommand: execCommand,
		flexDriver:  flexDriver,
		identity:    identity,
		runner:      exec.New(),
	}

	return provisioner
}

type flexProvisioner struct {
	client      kubernetes.Interface
	execCommand string
	flexDriver  string
	identity    types.UID
	runner      exec.Interface
}

var _ controller.Provisioner = &flexProvisioner{}


func (p *flexProvisioner) Provision(options controller.ProvisionOptions) (*v1.PersistentVolume, error) {

	annotations := make(map[string]string)
	annotations[annCreatedBy] = createdBy
	annotations[annProvisionerID] = string(p.identity)

	server := options.StorageClass.Parameters["server"]
	share := options.StorageClass.Parameters["share"]
	secretRef := options.StorageClass.Parameters["secretRef"]

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:        options.PVName,
			Labels:      map[string]string{},
			Annotations: annotations,
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: *options.StorageClass.ReclaimPolicy,
			AccessModes:                   options.PVC.Spec.AccessModes,
			Capacity: v1.ResourceList{
				v1.ResourceStorage: options.PVC.Spec.Resources.Requests[v1.ResourceStorage],
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				FlexVolume: &v1.FlexPersistentVolumeSource{
					Driver: p.flexDriver,
					Options: map[string]string{
						"server": server,
						"share":  "/" + share,
					},
					ReadOnly: false,
					SecretRef: &v1.SecretReference{
						Name: secretRef,
					},
				},
			},
		},
	}

	return pv, nil
}
