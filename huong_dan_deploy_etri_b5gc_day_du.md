# Hướng dẫn deploy ETRI B5GC trên Kubernetes (Minikube, 1 VM)

## 1. Mô hình triển khai

Mô hình này dùng:

- 1 máy ảo Ubuntu
- Kubernetes chạy bằng **Minikube**
- Runtime build image dùng **Docker**
- **PRAN chạy trong Kubernetes**
- **UPF chạy ngoài cluster**
- Toàn bộ manifest dùng namespace: **etri6g**

Luồng tổng quát:

- Các NF core như controller, gateway, UDR, NSM, AUSF, UDM, DAMF, SMF, AMF chạy trong cluster
- PRAN cũng chạy trong cluster
- UPF chạy ngoài cluster, đăng ký vào gateway thông qua địa chỉ được expose ra host

---

## 2. Chuẩn bị VM

Đăng nhập vào VM Ubuntu, sau đó cập nhật hệ thống:

```bash
sudo apt update
sudo apt upgrade -y
```

Cài các gói cơ bản:

```bash
sudo apt install -y curl wget git make gcc g++ unzip conntrack apt-transport-https ca-certificates software-properties-common
```

---

## 3. Cài Docker

Cài Docker:

```bash
sudo apt install -y docker.io
```

Bật Docker:

```bash
sudo systemctl enable docker
sudo systemctl start docker
```

Cho user hiện tại dùng Docker mà không cần sudo:

```bash
sudo usermod -aG docker $USER
```

Áp dụng group ngay trong phiên hiện tại:

```bash
newgrp docker
```

Kiểm tra:

```bash
docker --version
docker ps
```

---

## 4. Cài kubectl

Tải kubectl:

```bash
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
```

Cài vào hệ thống:

```bash
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
```

Kiểm tra:

```bash
kubectl version --client
```

---

## 5. Cài Minikube

Tải Minikube:

```bash
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
```

Cài Minikube:

```bash
sudo install minikube-linux-amd64 /usr/local/bin/minikube
```

Kiểm tra:

```bash
minikube version
```

---

## 6. Start Minikube

Vì đang chạy trên 1 VM nên dùng Docker driver:

```bash
minikube start --driver=docker
```

Kiểm tra node:

```bash
kubectl get nodes
```

Bạn phải thấy node ở trạng thái `Ready`.

---

## 7. Cho Docker build image trực tiếp vào Minikube

Thiết lập môi trường Docker của Minikube:

```bash
eval $(minikube docker-env)
```

Kiểm tra:

```bash
docker images
```

Từ thời điểm này, các image build ra sẽ được Minikube dùng trực tiếp.

---

## 8. Vào repo và build image

Di chuyển vào repo:

```bash
cd ~/etriB5gc
```

Build toàn bộ image:

```bash
make docker
```

Nếu muốn build theo thư mục docker riêng:

```bash
cd docker
make all
cd ..
```

Kiểm tra image:

```bash
docker images | grep b5gc
```

Kỳ vọng thấy các image như:

- `b5gc-controller`
- `b5gc-gateway`
- `b5gc-nsm`
- `b5gc-ausf`
- `b5gc-udm`
- `b5gc-damf`
- `b5gc-smf`
- `b5gc-amf`
- `b5gc-pran`

---

## 9. Tạo namespace

Tạo namespace đúng với manifest của repo:

```bash
kubectl create namespace etri6g
```

Đặt namespace mặc định cho context hiện tại:

```bash
kubectl config set-context --current --namespace=etri6g
```

Kiểm tra:

```bash
kubectl get ns
```

---

## 10. Sửa manifest trước khi deploy

Đây là phần quan trọng nhất. Với bài toán chạy trên 1 VM, Minikube, PRAN trong cluster và UPF ngoài cluster, cần sửa một số manifest cho phù hợp.

---

## 10.1. Sửa file `k8s/daejeon/gateway.yaml`

Mở file:

```bash
nano k8s/daejeon/gateway.yaml
```

### 10.1.1. Sửa label `region`

Trong file gốc, phần selector là:

```yaml
selector:
  matchLabels:
    app: gateway
    region: daejeon
```

Nhưng phần template label lại đang là:

```yaml
labels:
  app: gateway
  region: seoul
```

Cần đổi thành:

```yaml
labels:
  app: gateway
  region: daejeon
```

### 10.1.2. Sửa `gateway.json` trong ConfigMap

Tìm phần dữ liệu cấu hình dạng JSON và sửa thành:

```json
{
  "registeredAddr": "192.168.1.100:7777",
  "controller": "10.100.100.10:8888",
  "labels": {
    "loc": "daejeon"
  }
}
```

### Giải thích

- `registeredAddr`: là địa chỉ để **UPF ngoài cluster** gọi vào gateway
- `controller`: vì gateway đang nằm **trong cluster**, nên nó nên gọi controller qua **ClusterIP**
- `10.100.100.10:8888` là địa chỉ service của controller theo manifest

Lưu file sau khi sửa.

---

## 10.2. Sửa file `k8s/seoul/nsm.yaml`

Mở file:

```bash
nano k8s/seoul/nsm.yaml
```

Tìm đoạn:

```json
"url": "mongodb://192.168.0.121:30001"
```

Đổi thành:

```json
"url": "mongodb://10.100.100.100:27017"
```

### Giải thích

MongoDB chạy trong cluster, nên NSM phải truy cập MongoDB qua service nội bộ của Kubernetes.

Service của UDR/MongoDB theo manifest:

- ClusterIP: `10.100.100.100`
- Port: `27017`

Lưu file sau khi sửa.

---

## 10.3. Sửa file `k8s/daejeon/smf.yaml`

Mở file:

```bash
nano k8s/daejeon/smf.yaml
```

Tìm dòng:

```yaml
replicas: 5
```

Đổi thành:

```yaml
replicas: 1
```

### Giải thích

Chỉ dùng 1 VM nên không cần 5 SMF replica.

Lưu file.

---

## 10.4. Sửa file `k8s/daejeon/amf-10-100.yaml`

Mở file:

```bash
nano k8s/daejeon/amf-10-100.yaml
```

Tìm dòng:

```yaml
replicas: 10
```

Đổi thành:

```yaml
replicas: 1
```

### Giải thích

10 AMF là quá nặng đối với mô hình 1 VM.

Lưu file.

---

## 11. Tạo manifest cho PRAN trong Kubernetes

Repo không có sẵn file `pran.yaml`, nên cần tự tạo.

Tạo file mới:

```bash
nano k8s/daejeon/pran.yaml
```

Dán toàn bộ nội dung sau:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pran-dep
  namespace: etri6g
spec:
  replicas: 1
  selector:
    matchLabels:
      app: pran
  template:
    metadata:
      labels:
        app: pran
        plmnId: 208-93
    spec:
      volumes:
        - name: pran-config
          configMap:
            name: pran-config
      containers:
        - name: pran
          image: b5gc-pran:latest
          imagePullPolicy: IfNotPresent
          command: ["./pran"]
          args: ["-c", "/etc/config/pran.json"]
          volumeMounts:
            - mountPath: /etc/config
              name: pran-config
          ports:
            - containerPort: 9001
            - containerPort: 7001
            - containerPort: 38412
              protocol: SCTP
          env:
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace

---
apiVersion: v1
kind: ConfigMap
metadata:
  name: pran-config
  namespace: etri6g
data:
  pran.json: |
    {
      "nfSelection": {
        "gnb-loc": "daejeon"
      },
      "nasSplit": false,
      "transportNetworks": ["tran1"],
      "plmnId": {
        "mcc": "208",
        "mnc": "93"
      },
      "amfRegion": 10,
      "mesh": {
        "registrar": "10.100.100.1",
        "bind": "0.0.0.0",
        "labels": {
          "plmnId": "208-93",
          "tac": "001",
          "app": "pran"
        }
      },
      "ngapBinds": ["0.0.0.0"],
      "ngapPort": 38412
    }

---
apiVersion: v1
kind: Service
metadata:
  name: pran-svc
  namespace: etri6g
spec:
  selector:
    app: pran
  type: NodePort
  ports:
    - name: sbi
      protocol: TCP
      port: 9001
      targetPort: 9001
    - name: ngap
      protocol: SCTP
      port: 38412
      targetPort: 38412
      nodePort: 30500
```

Lưu file.

---

## 12. Build binary UPF để chạy ngoài cluster

Vì UPF không chạy trong Kubernetes, cần build binary riêng:

```bash
make upf
```

Kiểm tra:

```bash
ls bin/upf
```

---

## 13. Tạo config cho UPF chạy ngoài cluster

Tạo file mới:

```bash
nano config/upf-local.json
```

Dán nội dung sau:

```json
{
  "plmnId": {
    "mcc": "208",
    "mnc": "93"
  },
  "slices": [
    {
      "sd": "010203",
      "sst": 1
    },
    {
      "sd": "543210",
      "sst": 1
    }
  ],
  "isAnchor": true,
  "dnnList": [],
  "dnaiList": [],
  "iflist": [
    {
      "ip": "192.168.1.100",
      "mtu": 1400,
      "name": "tran1"
    }
  ],
  "mesh": {
    "registrar": "192.168.49.2:30007",
    "bind": "192.168.1.100",
    "labels": {
      "plmnId": "208-93",
      "app": "upf"
    }
  }
}
```

Lưu file.

### Giải thích cấu hình

- `iflist.ip`: địa chỉ host hoặc interface mà UPF dùng để bind
- `mesh.bind`: địa chỉ mà UPF bind khi chạy ngoài cluster
- `mesh.registrar`: địa chỉ mà UPF dùng để đăng ký vào gateway đã expose

Trong phần ghi chú ban đầu có nhắc đến `192.168.1.100:7777` là cổng gateway được expose ra host. Khi triển khai thực tế, bạn cần kiểm tra lại chính xác địa chỉ đang được port-forward hoặc service expose để đồng bộ giữa:

- file `gateway.yaml`
- file `upf-local.json`
- IP thật của VM / Minikube / host

---

## 14. Tạo thư mục dữ liệu cho MongoDB

Trước khi apply `udr.yaml`, tạo thư mục dữ liệu cho hostPath:

```bash
sudo mkdir -p /data/db
sudo chmod 777 /data/db
```

---

## 15. Deploy theo đúng thứ tự

Thực hiện deploy theo đúng thứ tự để tránh lỗi dependency.

---

## 15.1. Deploy Controller

```bash
kubectl apply -f k8s/seoul/controller.yaml
```

Kiểm tra:

```bash
kubectl get pods
kubectl get svc
```

---

## 15.2. Deploy Gateway

```bash
kubectl apply -f k8s/daejeon/gateway.yaml
```

Kiểm tra:

```bash
kubectl get pods
kubectl get svc
```

Kỳ vọng thấy service như:

- `ctrl-svc`
- `gateway-svc`

---

## 15.3. Expose gateway ra host bằng port-forward

Mở một terminal SSH khác hoặc dùng `tmux` / `screen`, sau đó chạy:

```bash
kubectl port-forward deployment/gateway-dep 7777:7777 --address 0.0.0.0 -n etri6g
```

Giữ lệnh này chạy liên tục.

### Ý nghĩa

- Pod trong cluster gọi gateway qua service nội bộ, ví dụ `10.100.100.1`
- UPF ngoài cluster gọi gateway qua địa chỉ host, ví dụ `192.168.1.100:7777`

---

## 15.4. Deploy UDR / MongoDB

```bash
kubectl apply -f k8s/seoul/udr.yaml
```

Kiểm tra:

```bash
kubectl get pods
kubectl get pvc
kubectl get pv
```

---

## 15.5. Deploy NSM

```bash
kubectl apply -f k8s/seoul/nsm.yaml
```

Kiểm tra log:

```bash
kubectl logs deployment/nsm-dep -n etri6g
```

---

## 15.6. Deploy service definitions

```bash
kubectl apply -f k8s/seoul/services.yaml
```

---

## 15.7. Deploy AUSF

```bash
kubectl apply -f k8s/seoul/ausf.yaml
```

---

## 15.8. Deploy UDM

```bash
kubectl apply -f k8s/seoul/udm.yaml
```

---

## 15.9. Deploy DAMF

```bash
kubectl apply -f k8s/daejeon/damf.yaml
```

---

## 15.10. Deploy SMF

```bash
kubectl apply -f k8s/daejeon/smf.yaml
```

---

## 15.11. Deploy AMF

```bash
kubectl apply -f k8s/daejeon/amf-10-100.yaml
```

---

## 15.12. Deploy PRAN

```bash
kubectl apply -f k8s/daejeon/pran.yaml
```

---

## 16. Kiểm tra toàn bộ pod và service

Kiểm tra pod:

```bash
kubectl get pods -n etri6g -o wide
```

Kiểm tra service:

```bash
kubectl get svc -n etri6g
```

Kỳ vọng có các pod sau:

- `ctrl-dep`
- `gateway-dep`
- `mongo-db`
- `nsm-dep`
- `ausf-dep`
- `udm-dep`
- `damf-dep`
- `smf-slice1-dep`
- `amf-10-100-dep`
- `pran-dep`

---

## 17. Chạy UPF ngoài cluster

Sau khi gateway đã được port-forward và các NF cần thiết đã lên đầy đủ, chạy UPF ngoài cluster:

```bash
./bin/upf -c config/upf-local.json
```

Nếu binary chưa có quyền thực thi:

```bash
chmod +x ./bin/upf
./bin/upf -c config/upf-local.json
```

---

## 18. Các file cấu hình đã sửa hoặc tạo mới

Trong quá trình deploy này, các file config / manifest liên quan gồm:

### Các file đã sửa
- `k8s/daejeon/gateway.yaml`
- `k8s/seoul/nsm.yaml`
- `k8s/daejeon/smf.yaml`
- `k8s/daejeon/amf-10-100.yaml`

### Các file tạo mới
- `k8s/daejeon/pran.yaml`
- `config/upf-local.json`

### Các file được apply khi deploy
- `k8s/seoul/controller.yaml`
- `k8s/daejeon/gateway.yaml`
- `k8s/seoul/udr.yaml`
- `k8s/seoul/nsm.yaml`
- `k8s/seoul/services.yaml`
- `k8s/seoul/ausf.yaml`
- `k8s/seoul/udm.yaml`
- `k8s/daejeon/damf.yaml`
- `k8s/daejeon/smf.yaml`
- `k8s/daejeon/amf-10-100.yaml`
- `k8s/daejeon/pran.yaml`

---

## 19. Checklist nhanh

Sau khi hoàn tất, cần đảm bảo:

- Docker chạy bình thường
- Minikube lên thành công
- Node ở trạng thái `Ready`
- Đã chạy `eval $(minikube docker-env)`
- Docker images đã được build thành công
- Namespace `etri6g` đã được tạo
- Các manifest đã sửa đúng IP và replica
- PRAN manifest đã được tạo
- MongoDB đã mount được thư mục `/data/db`
- Gateway đang được port-forward ở cổng `7777`
- Tất cả pod chính đã ở trạng thái `Running`
- UPF ngoài cluster chạy được bằng file `config/upf-local.json`

---

## 20. Ghi chú quan trọng về IP

Trong hướng dẫn này có các IP ví dụ như:

- `192.168.1.100`
- `192.168.49.2`
- `10.100.100.1`
- `10.100.100.10`
- `10.100.100.100`

Khi triển khai thực tế, cần kiểm tra kỹ:

```bash
minikube ip
kubectl get svc -n etri6g
ip a
```

Nếu IP thật của VM hoặc Minikube khác với ví dụ trong file, bạn phải sửa lại đồng bộ trong:

- `k8s/daejeon/gateway.yaml`
- `config/upf-local.json`

Nếu không đồng bộ IP, UPF ngoài cluster sẽ không đăng ký được hoặc gateway/controller sẽ không liên lạc đúng.

---

## 21. Kết luận

Quy trình này phù hợp cho mô hình demo hoặc lab nhỏ:

- 1 VM
- Minikube
- PRAN trong cluster
- UPF ngoài cluster

Nếu mở rộng sang nhiều VM hoặc cloud server, nên chuyển sang hướng:

- Terraform để tạo máy
- Ansible để cài đặt và cấu hình
- kubeadm để dựng cluster nhiều node
- Tự động hóa deploy toàn bộ NF bằng playbook hoặc script
