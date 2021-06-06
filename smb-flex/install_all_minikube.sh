#!/bin/sh

eval $(minikube docker-env)
./install_driver.sh
./install_example.sh

