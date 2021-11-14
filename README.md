# Webapp Kubernetes Operator Example with TLS

## Installation guide
First Docker and kubectl need to be installed:
- [Docker installation instructions](https://docs.docker.com/engine/install/)
- [kubectl installation instructions](https://kubernetes.io/docs/tasks/tools/)

A working Kubernetes environment is also needed, for testing purposes k3s can be used. Install it with the following parameters:
```
curl -sfL https://get.k3s.io | sh -s - --write-kubeconfig-mode 644 --disable traefik
```
Next add the needed other extensions (ingress may differ in case of production grade environments):
```
kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.6.0/cert-manager.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/static/provider/baremetal/deploy.yaml
```
After that apply the ingress patch:
```
kubectl patch deployment ingress-nginx-controller -n ingress-nginx --patch "$(cat ingress_local_patch.yaml)"
```
Then deploy the application by using an image from [here](https://github.com/Kuvesz/operator-test/pkgs/container/operator-test) and running make with the following parameteres:
```
make deploy IMG=[IMAGE NAME HERE]
```
After this a yaml is needed to configure and run the service. An example of that can be found under `config/samples`.
