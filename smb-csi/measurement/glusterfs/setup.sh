#!/bin/sh

# GlusterFS only working on kubernetes version below 1.20.0
minikube start --kubernetes-version=v1.19.4
# Installing GlusterFS deps into minikube
minikube ssh -- sudo apt update
minikube ssh -- sudo apt install glusterfs-client -y
# Installing driver
kubectl apply -f rbac.yaml
kubectl apply -f glusterfs-daemonset.yaml
kubectl apply -f deployment.yaml
kubectl apply -f storageclass.yaml
