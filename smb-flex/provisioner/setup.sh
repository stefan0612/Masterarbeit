#!/bin/sh

eval $(minikube docker-env)
make
docker build -t seitenbau/smb-flex-provisioner .
make clean