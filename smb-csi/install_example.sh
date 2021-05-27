#Installing Secrets
kubectl apply -f deploy/example/secret/smb-secret.yml

# Creating SMB-Server (Depends on Cluster Configs, thus must be created after Cluster Configs)
kubectl apply -f deploy/example/server/smb-server.yml

# Create Base Classes for Storage and Snapshot
kubectl apply -f deploy/example/storage/smb-storage-class.yml
kubectl apply -f deploy/example/snapshot/smb-snap-class.yml

# Creating PVC
kubectl apply -f deploy/example/storage/smb-pvc.yml

# Creating Client consuming the PVC
kubectl apply -f deploy/example/client/smb-client.yml

while kubectl describe pod smb-client | grep Pending;
do
  echo "Waiting for Smb-Client to Run"
  sleep 5
done

# Creating Snapshot from given PVC
kubectl apply -f deploy/example/snapshot/smb-snapshot.yml

while kubectl describe volumesnapshot smb-snap | grep false;
do
  echo "Waiting for Snapshot to be Created"
  sleep 5
done

# Restoring PVC from given Snapshot
kubectl apply -f deploy/example/storage/smb-pvc-snap.yml

# Cloning existing PVC
kubectl apply -f deploy/example/storage/smb-pvc-cloned.yml
