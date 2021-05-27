# Creating Controller Server
kubectl delete -f deploy/driver/controller/controller-server.yml
kubectl delete -f deploy/driver/controller/rbac-controller-server.yml

# Creating Node Server
kubectl delete -f deploy/driver/node/node-server.yml
kubectl delete -f deploy/driver/node/rbac-node-server.yml
