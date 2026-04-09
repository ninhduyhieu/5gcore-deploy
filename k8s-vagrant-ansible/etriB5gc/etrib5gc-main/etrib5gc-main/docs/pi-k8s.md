Setup k8s Pi

# Flash Rasbian Os lite version

# Enable ssh server

 - mount the bootfs folder
 - create `userconf.txt` with user:password; password generated with `openssl
   passwd -6`
 - create an `ssh` file on the bootfs folder

Boot up the Pi and check for its IP, now you can access it using ssh

# Install packages: 
 - vim, git, curl
 - for go lang, download binary then

```bash
	rm -rf /usr/local/go && tar -C /usr/local -xzf go1.24.4.linux-amd64.tar.gz
```
# Install docker

```bash
curl -sSL https://get.docker.com | sh
```

Add user to docker group

```bash
	sudo usermod -aG docker $USER
```
logout/in for the change to take effect, now you can test docker:
```bash
	docker run hello-world
```
# Install minikube

```bash
	curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube_latest_arm64.deb
	sudo dpkg -i minikube_latest_arm64.deb
```
Now install `kubctrl`:
```bash
	curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/arm64/kubectl"
	sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
```

 
