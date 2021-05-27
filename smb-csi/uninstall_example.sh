# Deleting Client consuming the PVC and Snapshot
kubectl delete -f deploy/example/client/smb-client.yml
kubectl delete -f deploy/example/snapshot/smb-snapshot.yml

# Deleting PVCs
kubectl delete -f deploy/example/storage/smb-pvc.yml
kubectl delete -f deploy/example/storage/smb-pvc-snap.yml
kubectl delete -f deploy/example/storage/smb-pvc-cloned.yml

# Deleting SMB Server
kubectl delete -f deploy/example/server/smb-server.yml

# Deleting Base Classes
kubectl delete -f deploy/example/storage/smb-storage-class.yml
kubectl delete -f deploy/example/snapshot/smb-snap-class.yml

# Deleting Secrets
kubectl delete -f deploy/example/secret/smb-secret.yml
