kubectl apply -f ../../deploy/example/secret/smb-secret.yml
kubectl apply -f ../../deploy/example/server/smb-server.yml

(cd ../../; ./install_driver.sh)

# Create Base Classes for Storage and Snapshot
kubectl apply -f ../../deploy/example/storage/smb-storage-class.yml
kubectl apply -f ../../deploy/example/snapshot/smb-snap-class.yml
