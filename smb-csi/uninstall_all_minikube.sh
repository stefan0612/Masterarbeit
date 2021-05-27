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

./uninstall_example.sh
./uninstall_driver.sh

