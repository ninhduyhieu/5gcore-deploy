# 5G Core Deploy

Automated deployment of a K3s Kubernetes cluster for 5G Core using Vagrant and Ansible.

### [Vagrant](./Vagrantfile) VMs
| Hostname | IP | CPU | RAM |
|----------|-----|-----|-----|
| k3s-server-1 | 192.168.58.11 | 4 | 4GB |
| k3s-agent-1 | 192.168.58.12 | 4 | 2GB |
| k3s-agent-2 | 192.168.58.13 | 4 | 2GB |

### NOTE: use this [box](https://drive.google.com/file/d/1z-JIrnqV2uSxIK9hmDth7D8YGC0TYqsi/view?usp=sharing) to boot vms
```bash
vagrant box add ubuntu2204 ubuntu2204.box

# then boot
vagrant up
```

### Ansible [Roles](./roles/) (in playbook order)
- `common` - apt update, base packages, hosts file
- `docker_install` - Docker runtime
- `k3s_install` - K3s server with embedded etcd
- `k3s_kubeconfig` - Fetch kubeconfig from master
- `k3s_agent` - K3s agents joining cluster
- `k3s_calico` - Calico CNI plugin

## Prerequisites

- [VirtualBox](https://www.virtualbox.org/)
- [Vagrant](https://www.vagrantup.com/)
- [Ansible](https://www.ansible.com/)

> A Nix flake is provided for convenience — enter by `direnv allow` to get Ansible installed automatically.
```bash
direnv allow  # Activates Nix flake with Ansible, kubectl, k9s
```

### NOTE: to use kubectl, have to set env `export KUBECONFIG="$PWD/kubeconfig"`

## Apply Ansible to create cluster
```
ansible-playbook -i inventory/hosts.yaml playbook.yml
```

## Docker build images

```
# On your host PC
cd /path/to/etrib5gc/docker

# Build base image first
make base

# Build all NF images
make all

# Verify images are built
docker images | grep b5gc

# Save images to tar files
docker save b5gc-controller:latest -o b5gc-controller.tar
docker save b5gc-gateway:latest -o b5gc-gateway.tar
docker save b5gc-nsm:latest -o b5gc-nsm.tar
docker save b5gc-ausf:latest -o b5gc-ausf.tar
docker save b5gc-udm:latest -o b5gc-udm.tar
docker save b5gc-pcf:latest -o b5gc-pcf.tar
docker save b5gc-damf:latest -o b5gc-damf.tar
docker save b5gc-amf:latest -o b5gc-amf.tar
docker save b5gc-smf:latest -o b5gc-smf.tar

scp b5gc-*.tar vagrant@192.168.58.11:/tmp/
scp b5gc-*.tar vagrant@192.168.58.12:/tmp/
scp b5gc-*.tar vagrant@192.168.58.13:/tmp/

ssh vagrant@192.168.58.11 "for f in /tmp/b5gc-*.tar; do sudo k3s ctr images import \$f; done"
ssh vagrant@192.168.58.12 "for f in /tmp/b5gc-*.tar; do sudo k3s ctr images import \$f; done"
ssh vagrant@192.168.58.13 "for f in /tmp/b5gc-*.tar; do sudo k3s ctr images import \$f; done"
```

## Create Namespace
```bash
kubectl create namespace etri6g
kubectl config set-context --current --namespace=etri6g
```

## Start deployments [k8s](./k8s/)
```bash
kubectl apply -f ./k8s/ctrl.yaml
kubectl apply -f ./k8s/gw.yaml
kubectl apply -f ./k8s/nsm.yaml
kubectl apply -f ./k8s/mongo.yaml
kubectl apply -f ./k8s/deploy-other.yaml
```
