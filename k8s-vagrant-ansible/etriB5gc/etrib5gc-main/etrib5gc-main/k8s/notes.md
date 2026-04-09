## Using docker local repo in minikube

First, start minikube

```bash
minikube start
```

then set the docker environment with

```bash
eval $(minikube docker-env)
```

now build docker image with docker command, the built image can be pull by minikube locally

## Expose Pod to applications on host machine


```bash
kubectrl port-forward deployments/ctrl 8888:8888 --address 192.168.0.9
```

## get minikube virtual machine ip address

```bash
minikube ip
```
Default minikube ip address is 192.168.49.2

## configure controller:

Define controller service using NodePort type. The controller is exposed on the
minikube's ip (192.168.49.2:30331). Here 30331 is the nodePort defined in the
service spec manifest

## configure gateway


## running b5gcore

- deploy controller
- expose controller
- deploy gateway 
- expose gateway (using port-forward). The host machine:7777 will become registrar IP for external NFs
- deploy nsm
- deploy remaining NFs
- run pran outside of the cluster
- run upf outside of the cluster 
- run UERANSIM. Note that gnb (nr-gnb) and UPF should not be on the same machine; run ue emulator (nr-ue) with sudo

## expose the controller/gateway to public network

 ```bash
	sudo iptables -t nat -A PREROUTING -p tcp --dport 8888 -j DNAT --to-destnation 192.168.49.2:30331
	sudo iptables -I FORWARDING -p tcp -d 192.168.49.2 -j ACCEPT
	sudo iptables -A POSTROUTING -p tcp -d 192.168.49.2 -j MASQUERADE
 ``` 


## deploy PRAN on k8s
The procedure is similar to any other network functions, but we need to expose the SCTP.
Deploy a PRAN service of type NodePort and assign a port in the NodePort range (38412 is outside of the range). Let say the port is 30500.
If your gnB is running on the same host with the minikube, it can access to the PRAN with minikubeIP:30500. Otherwise, you will need port forwarding.
```
	sudo iptables -t nat -I PREROUTING -p sctp -d HOST_IP --dport 38412 -j DNAT --to-destination MINIKUBE_IP:30500
	sudo iptables -I FORWARD -p sctp -d MINIKUBE_IP -j ACCEPT
```
