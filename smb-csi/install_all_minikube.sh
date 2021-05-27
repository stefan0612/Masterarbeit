#!/bin/sh

if minikube status | grep Stopped;
then
  echo "\e[91mMinikube not Running\e[0m"
  echo "\e[93mStarting Minikube\e[0m"
  if minikube start;
  then
    echo "\e[92mMinikube started successfully\e[0m"
    minikube addons enable volumesnapshots
  else
    echo "\e[91mMinikube cannot be started\e[0m"
    exit 1
  fi
else
  minikube addons enable volumesnapshots
fi

# Build driver-docker-image and deploy it to minikube
eval $(minikube docker-env)
make
docker build -t seitenbau/smb-csi-driver .
make clean

./install_driver.sh
./install_example.sh
