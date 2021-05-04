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

package main

import (
	"flag"
	"strings"

	vol "github.com/volume"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"sigs.k8s.io/sig-storage-lib-external-provisioner/controller"
)

var (
	provisioner = flag.String("provisioner", "seitenbau/smb-flex-provisioner", "Name of the provisioner. The provisioner will only provision volumes for claims that request a StorageClass with a provisioner field set equal to this name.")
	master      = flag.String("master", "", "Master URL to build a client config from. Either this or kubeconfig needs to be set if the provisioner is being run out of cluster.")
	kubeconfig  = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Either this or master needs to be set if the provisioner is being run out of cluster.")
	execCommand = flag.String("execCommand", "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/seitenbau~cifs/cifs", "The provisioner executable.")
	flexDriver  = flag.String("flexDriver", "seitenbau/cifs", "The FlexVolume driver.")
)

func main() {
	flag.Set("logtostderr", "true")
	klog.InitFlags(nil)
	flag.Parse()

	if errs := validateProvisioner(*provisioner, field.NewPath("provisioner")); len(errs) != 0 {
		klog.Fatalf("Invalid provisioner specified: %v", errs)
	}
	klog.Infof("Provisioner %s specified", *provisioner)

	if execCommand == nil {
		klog.Fatalf("Invalid flags specified: must provide provisioner exec command")
	}

	if flexDriver == nil || *flexDriver == "" {
		klog.Fatalf("Invalid flags specified: must provide FlexVolume driver")
	}

	var config *rest.Config
	var err error
	if *master != "" || *kubeconfig != "" {
		klog.Infof("Either master or kubeconfig specified. building kube config from that..")
		config, err = clientcmd.BuildConfigFromFlags(*master, *kubeconfig)
	} else {
		klog.Infof("Building kube configs for running in cluster...")
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		klog.Fatalf("Failed to create config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create client: %v", err)
	}

	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		klog.Fatalf("Error getting server version: %v", err)
	}

	flexProvisioner := vol.NewFlexProvisioner(clientset, *execCommand, *flexDriver)

	// Start the provision controller which will dynamically provision SMB PVs
	pc := controller.NewProvisionController(
		clientset,
		*provisioner,
		flexProvisioner,
		serverVersion.GitVersion,
	)

	pc.Run(wait.NeverStop)
}

// validateProvisioner tests if provisioner is a valid qualified name.
func validateProvisioner(provisioner string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(provisioner) == 0 {
		allErrs = append(allErrs, field.Required(fldPath, provisioner))
	}
	if len(provisioner) > 0 {
		for _, msg := range validation.IsQualifiedName(strings.ToLower(provisioner)) {
			allErrs = append(allErrs, field.Invalid(fldPath, provisioner, msg))
		}
	}
	return allErrs
}
