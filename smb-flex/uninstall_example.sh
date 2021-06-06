#!/bin/sh
kubectl delete -f deploy/example/client/smb-client.yml
kubectl delete -f deploy/example/storage/smb-pvc.yml
kubectl delete -f deploy/example/server/smb-server.yml
kubectl delete -f deploy/example/storage/smb-storage-class.yml
kubectl delete -f deploy/example/secret/smb-secret.yml
