kubectl apply -f ../../deploy/example/secret/smb-secret.yml
kubectl apply -f ../../deploy/example/server/smb-server.yml

(cd ../../; ./install_driver_simple.sh)

# Create Base Classes for Storage
kubectl apply -f ../../deploy_simple/smb-storage-class.yml
