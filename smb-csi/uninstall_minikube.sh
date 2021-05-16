#!/bin/sh

if minikube status | grep Stopped;
then
  echo "\e[91mMinikube not Running\e[0m"
  echo "\e[93mStarting Minikube\e[0m"
  if minikube start;
  then
    echo "\e[92mMinikube started successfully\e[0m"
  else
    echo "\e[91mMinikube cannot be started\e[0m"
    exit 1
  fi
fi

kubectl delete -f deploy/test/snapshot/smb-snapshot.yml
# Deleting Client Chain (Depends on Servers, thus must be delete before Servers)
kubectl delete -f deploy/test/client/smb-client.yml
kubectl delete -f deploy/storage/smb-pvc.yml
kubectl delete -f deploy/storage/smb-sc.yml

# Deleting Servers (Depends on Cluster Configs, thus must be deleted before Cluster Configs)
kubectl delete -f deploy/controller-server.yml
kubectl delete -f deploy/node-server.yml
kubectl delete -f deploy/test/server/smb-server.yml

# Deleting Cluster Configurations
#kubectl delete -f deploy/driverConfig/smb-driver.yml
kubectl delete -f deploy/rbac-controller-server.yml
kubectl delete -f deploy/secret/smb-secret.yml

