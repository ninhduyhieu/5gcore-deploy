## Build docker image for Minikube

Run:
```bash
eval $(minikube docker-env)
```

## Expose node port service on host IP

let say we have a controller deploy in K8s in nodePort service type.
First, we get the url of the service:

```bash
minikube service <service-name> --url
```

The output is the exposed IP:port (service-url) of the service where IP is the minikube node's IP and port is the nodePort of the service.

Create following rules for IP table:

```bash
iptables -t nat -A PREROUTING -dport 8888 -j DNAT --to-destination <minikube IP:nodePort>
iptables -t nat -A FORWARD -d <minikube IP> -dport <nodePort> -j ACCEPT 
```
