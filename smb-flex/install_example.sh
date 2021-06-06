#!/bin/sh

kubectl apply -f deploy/example/secret/smb-secret.yml
kubectl apply -f deploy/example/server/smb-server.yml
kubectl apply -f deploy/example/storage/smb-storage-class.yml
kubectl apply -f deploy/example/storage/smb-pvc.yml
kubectl apply -f deploy/example/client/smb-client.yml

