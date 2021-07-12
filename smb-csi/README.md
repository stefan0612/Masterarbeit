# Dynamic volume provisioning of third-party storages in kubernetes clusters

## Installation
Run
```
./install_all_minikube.sh
```
to install a whole example including the driver, server, client and example-snapshots.

Run
```
./install_driver.sh
```
for only installing the driver, including the node-server and controller-server.

## Setup

### Tested installation setup
* Minikube version: 1.22.0
* Kubernetes version 1.21.2 
* OS: Ubuntu 20.04
* GO version: 1.13.8

### Measurement hardware setup
* CPU: Intel Core i7-10700k
* RAM: 32GB
