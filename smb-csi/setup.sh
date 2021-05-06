#!/bin/sh

eval $(minikube docker-env)
make
docker build -t seitenbau/smb-csi-driver .
make clean
