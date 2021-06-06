#!/bin/sh

kubectl delete -f deploy/driver/installer/smb-driver-installer.yml

kubectl delete -f deploy/driver/provisioner/rbac-smb-provisioner.yml
kubectl delete -f deploy/driver/provisioner/smb-provisioner.yml
