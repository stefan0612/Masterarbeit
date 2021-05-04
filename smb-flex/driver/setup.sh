#!/bin/sh

eval $(minikube docker-env)
make
docker build -t seitenbau/smb-flexdriver-installer .
make clean