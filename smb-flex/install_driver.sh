#!/bin/sh

(
  cd driver || return
  ./setup.sh
)
kubectl apply -f deploy/driver/installer/smb-driver-installer.yml

(
  cd provisioner || return
  ./setup.sh
)
kubectl apply -f deploy/driver/provisioner/rbac-smb-provisioner.yml
kubectl apply -f deploy/driver/provisioner/smb-provisioner.yml


