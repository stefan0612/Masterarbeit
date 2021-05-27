make
# Image 'seitenbau/smb-csi-driver' needs to be deployed to dockerhub before usage
# Alternatively build image directly into kubernetes env and pull image locally
docker build -t seitenbau/smb-csi-driver .
make clean

# Creating Controller Server
kubectl apply -f deploy/driver/controller/rbac-controller-server.yml
kubectl apply -f deploy/driver/controller/controller-server.yml

# Creating Node Server
kubectl apply -f deploy/driver/node/rbac-node-server.yml
kubectl apply -f deploy/driver/node/node-server.yml
