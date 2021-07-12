#Installing Secrets
minikube addons enable volumesnapshots
kubectl apply -f ../deploy/example/secret/smb-secret.yml

# Creating SMB-Server (Depends on Cluster Configs, thus must be created after Cluster Configs)
kubectl apply -f ../deploy/example/server/smb-server.yml

# Create Base Classes for Storage and Snapshot
kubectl apply -f ../deploy/example/storage/smb-storage-class.yml
kubectl apply -f ../deploy/example/snapshot/smb-snap-class.yml
