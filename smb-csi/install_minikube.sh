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

# Build driver-docker-image and deploy it to minikube
eval $(minikube docker-env)
make
docker build -t seitenbau/smb-csi-driver .
make clean

# Creating Cluster Configurations
kubectl apply -f deploy/rbac-controller-server.yml
kubectl apply -f deploy/driverConfig/smb-driver.yml
kubectl apply -f deploy/secret/smb-secret.yml

# Creating Servers (Depends on Cluster Configs, thus must be created after Cluster Configs)
kubectl apply -f deploy/test/server/smb-server.yml
kubectl apply -f deploy/node-server.yml
kubectl apply -f deploy/controller-server.yml

# Create Client Chain (Depends on Servers, thus must be created after Servers)
kubectl apply -f deploy/storage/smb-sc.yml
kubectl apply -f deploy/storage/smb-pvc.yml
kubectl apply -f deploy/test/client/smb-client.yml
