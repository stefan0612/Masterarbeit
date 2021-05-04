#!/bin/sh

eval $(minikube docker-env)
docker build -t seitenbau/smb-flexdriver-installer .
make clean
