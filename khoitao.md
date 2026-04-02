# Mô tả chi tiết phase khởi tạo

---

# 1) Flow thật sự khi bạn chạy từng lệnh Vagrant

Phần này giải thích khi bạn chạy từng lệnh `vagrant`, bên trong dự án thực sự sẽ có những bước nào diễn ra, Vagrant gọi gì, Ansible gọi gì, module/role nào chạy, và log/output có ý nghĩa ra sao.

---

## 1.1. Khi chạy `vagrant up k8s-master`

Đây là lệnh dùng để dựng máy ảo master và chạy toàn bộ các bước provision ban đầu cho node master.

## 1.1.1. Vagrant sẽ làm gì?

Khi chạy:

```bash
vagrant up k8s-master
```

Vagrant sẽ thực hiện lần lượt các việc sau:

1. Tạo máy ảo `k8s-master`
2. Gán network cho máy
3. Gắn thư mục source từ máy host vào trong VM tại `/vagrant`
4. Chạy shell provisioner khởi động ban đầu
5. Chạy `ansible_local` để cấu hình hệ thống
6. Chạy playbook dành riêng cho master để khởi tạo Kubernetes cluster

---

## 1.1.2. Các đầu mục log Vagrant thường xuất hiện

Bạn thường sẽ thấy dạng log như sau:

```text
==> k8s-master: Importing base box 'generic/ubuntu2204'...
==> k8s-master: Matching MAC address for NAT networking...
==> k8s-master: Setting the name of the VM: k8s-master
==> k8s-master: Clearing any previously set network interfaces...
==> k8s-master: Preparing network interfaces based on configuration...
    k8s-master: Adapter 1: nat
    k8s-master: Adapter 2: hostonly
==> k8s-master: Forwarding ports...
    k8s-master: 22 (guest) => 2222 (host)
==> k8s-master: Running 'pre-boot' VM customizations...
==> k8s-master: Booting VM...
==> k8s-master: Waiting for machine to boot...
==> k8s-master: Machine booted and ready!
==> k8s-master: Checking for guest additions in VM...
==> k8s-master: Mounting shared folders...
    k8s-master: D:/k8s-vagrant-ansible => /vagrant
==> k8s-master: Running provisioner: shell...
==> k8s-master: Running provisioner: ansible_local...
```

---

## 1.1.3. Giải thích ý nghĩa từng block log của Vagrant

### `Importing base box`
Vagrant đang lấy box nền, ví dụ Ubuntu 22.04.

### `Preparing network interfaces`
Vagrant cấu hình card mạng cho VM:
- NAT để VM có Internet
- host-only để các VM giao tiếp với nhau và host có thể SSH/debug

### `Forwarding ports`
Cổng SSH của guest được map ra host để bạn có thể SSH hoặc để Vagrant điều khiển VM.

### `Mounting shared folders`
Thư mục project trên host được mount vào `/vagrant` trong máy ảo. Đây là điểm rất quan trọng vì toàn bộ Ansible playbook, template, artifact đều dùng đường dẫn `/vagrant/...`.

### `Running provisioner: shell`
Chạy shell script đơn giản ban đầu, thường để chờ hệ thống ổn định hoặc chuẩn bị trước khi Ansible chạy.

### `Running provisioner: ansible_local`
Đây là điểm quan trọng nhất: Ansible không chạy từ host vào guest, mà chạy ngay **bên trong guest**. Vì vậy mới gọi là `ansible_local`.

---

## 1.1.4. Sau phần Vagrant, Ansible nào sẽ chạy?

Với `k8s-master`, thường sẽ chạy lần lượt:

1. `common.yml`
2. `master.yml`

Tức là:
- trước tiên cấu hình hệ thống chung
- sau đó khởi tạo master Kubernetes

---

## 1.2. `common.yml` trên master chạy gì?

Playbook `common.yml` gọi role `common`.

Role này chính là module cấu hình nền hệ thống để máy Ubuntu có thể chạy Kubernetes.

---

## 1.2.1. Các module/chức năng chính của role `common`

Role `common` thường đảm nhiệm các nhóm việc sau:

### 1. Tắt swap
Kubernetes yêu cầu swap phải tắt.

Task thường thấy:
- disable swap ngay lập tức
- comment hoặc remove swap trong `/etc/fstab`

### 2. Bật kernel modules cho networking/container
Các module như:
- `overlay`
- `br_netfilter`

được dùng để hỗ trợ networking của container và bridge traffic.

### 3. Áp dụng sysctl cho Kubernetes
Ví dụ:
- cho phép iptables nhìn thấy bridge traffic
- bật IP forwarding

### 4. Cài container runtime
Thường cài:
- `containerd`
- `docker.io`

Docker ở đây có thể hỗ trợ build image, còn runtime thực tế Kubernetes dùng thường là `containerd`.

### 5. Cấu hình containerd
- tạo file config mặc định
- chỉnh `SystemdCgroup = true`
- restart containerd

### 6. Cài Kubernetes packages
Thường gồm:
- `kubelet`
- `kubeadm`
- `kubectl`

### 7. Hold version gói
Giữ version không cho tự nâng cấp, để tránh lệch version giữa các node.

### 8. Enable kubelet
Bật service `kubelet` để sẵn sàng cho `kubeadm init` hoặc `kubeadm join`.

---

## 1.2.2. Output/log thường thấy của role `common`

```text
PLAY [Common setup for current node] *******************************************

TASK [Gathering Facts] *********************************************************
ok: [k8s-master]

TASK [common : Disable swap immediately] ***************************************
ok: [k8s-master]

TASK [common : Disable swap in fstab] ******************************************
changed: [k8s-master]

TASK [common : Load kernel modules] ********************************************
changed: [k8s-master]

TASK [common : Modprobe overlay] ***********************************************
ok: [k8s-master]

TASK [common : Modprobe br_netfilter] ******************************************
ok: [k8s-master]

TASK [common : Set sysctl params for Kubernetes] *******************************
changed: [k8s-master]

TASK [common : Apply sysctl] ***************************************************
ok: [k8s-master]

TASK [common : Install prerequisite packages] **********************************
changed: [k8s-master]

TASK [common : Ensure docker is enabled] ***************************************
ok: [k8s-master]

TASK [common : Create containerd config directory] *****************************
changed: [k8s-master]

TASK [common : Generate default containerd config if missing] ******************
changed: [k8s-master]

TASK [common : Enable SystemdCgroup in containerd] *****************************
changed: [k8s-master]

TASK [common : Restart containerd] *********************************************
changed: [k8s-master]

TASK [common : Add Kubernetes apt key] *****************************************
changed: [k8s-master]

TASK [common : Add Kubernetes repository] **************************************
changed: [k8s-master]

TASK [common : Install kubeadm kubelet kubectl] ********************************
changed: [k8s-master]

TASK [common : Hold kube packages] *********************************************
ok: [k8s-master] => (item=kubelet)
ok: [k8s-master] => (item=kubeadm)
ok: [k8s-master] => (item=kubectl)

TASK [common : Ensure kubelet enabled] *****************************************
ok: [k8s-master]
```

---

## 1.2.3. Giải thích output `ok`, `changed`, `failed`, `skipping`

### `ok`
Task chạy xong và không phải thay đổi gì thêm, hoặc task được đánh dấu là không thay đổi trạng thái.

### `changed`
Task đã làm thay đổi hệ thống. Ví dụ:
- sửa file config
- cài package
- restart service
- tạo thư mục

### `failed`
Task lỗi, playbook thường sẽ dừng tại đó nếu không có cơ chế bỏ qua lỗi.

### `skipping`
Task bị bỏ qua vì điều kiện `when` không thỏa.

---

## 1.3. `master.yml` trên master chạy gì?

`master.yml` gọi role `master_init`.

Đây là module chịu trách nhiệm biến `k8s-master` thành control-plane node của Kubernetes cluster.

---

## 1.3.1. Các chức năng chính của role `master_init`

### 1. Kiểm tra cluster đã init chưa
Nếu `/etc/kubernetes/admin.conf` đã tồn tại, có thể cluster đã được khởi tạo trước đó.

### 2. Reset trạng thái cũ nếu cần
Nếu đang dở dang hoặc cần init lại:
- `kubeadm reset -f`

### 3. Chạy `kubeadm init`
Đây là bước quan trọng nhất để tạo control-plane.

Ví dụ lệnh thực tế có thể là:

```bash
kubeadm init \
  --apiserver-advertise-address=192.168.56.10 \
  --pod-network-cidr=10.244.0.0/16 \
  --service-cidr=10.100.100.0/24 \
  --cri-socket unix:///run/containerd/containerd.sock \
  --kubernetes-version v1.29.15
```

### 4. Cấu hình kubeconfig cho user `vagrant`
Để user `vagrant` dùng `kubectl` dễ dàng.

### 5. Cài Flannel network
Flannel là CNI plugin giúp pod network hoạt động.

### 6. Chờ master sẵn sàng
Bao gồm:
- API server reachable
- flannel rollout xong
- node `Ready`
- system pods ready

### 7. Sinh `join-command.sh`
Tạo file lệnh để worker join vào cluster.

---

## 1.3.2. Output/log thường thấy của `master_init`

```text
PLAY [Initialize Kubernetes master] ********************************************

TASK [Gathering Facts] *********************************************************
ok: [k8s-master]

TASK [master_init : Check if cluster already initialized] **********************
ok: [k8s-master]

TASK [master_init : Reset old kubeadm state if needed] *************************
ok: [k8s-master]

TASK [master_init : Init cluster] **********************************************
changed: [k8s-master]

TASK [master_init : Create .kube directory for vagrant] ************************
changed: [k8s-master]

TASK [master_init : Copy admin.conf for vagrant user] **************************
changed: [k8s-master]

TASK [master_init : Wait until kube-apiserver is reachable] ********************
ok: [k8s-master]

TASK [master_init : Ensure kube-flannel namespace exists] **********************
ok: [k8s-master]

TASK [master_init : Apply Flannel manifest] ************************************
changed: [k8s-master]

TASK [master_init : Wait for Flannel DaemonSet to exist] ***********************
ok: [k8s-master]

TASK [master_init : Wait for Flannel rollout] **********************************
ok: [k8s-master]

TASK [master_init : Wait for all nodes to become Ready] ************************
ok: [k8s-master]

TASK [master_init : Create join command] ***************************************
ok: [k8s-master]

TASK [master_init : Save join command to shared folder] ************************
changed: [k8s-master]
```

---

## 1.3.3. Ý nghĩa output của `master_init`

### `Init cluster`
Nếu task này `changed`, nghĩa là cluster vừa được tạo mới.

### `Copy admin.conf`
Cho phép user `vagrant` dùng `kubectl`.

### `Apply Flannel manifest`
Cài CNI network. Nếu bước này lỗi, các pod sau này có thể không giao tiếp được.

### `Wait for Flannel rollout`
Chờ cho DaemonSet flannel chạy ổn định trên node.

### `Create join command`
Sinh ra lệnh `kubeadm join ...`, thường lưu vào:
- `/vagrant/ansible/join-command.sh`

File này là đầu vào rất quan trọng cho worker.

---

## 1.4. Khi chạy `vagrant up k8s-worker1`

Đây là lệnh dựng worker1 và cho node này tham gia cluster.

---

## 1.4.1. Vagrant sẽ làm gì?

Khi chạy:

```bash
vagrant up k8s-worker1
```

Vagrant sẽ:

1. Tạo VM worker1
2. Gắn IP cho worker1
3. Mount source vào `/vagrant`
4. Chạy `common.yml`
5. Chạy `worker.yml`
6. Có thể chạy thêm `upf.yml` tùy cấu hình

---

## 1.4.2. Các module Ansible thường chạy trên worker1

### `common`
Cấu hình OS, containerd, Kubernetes packages giống master.

### `worker_join`
Cho worker join vào cluster.

### `upf`
Nếu dự án bật phần external UPF, role này sẽ lo cấu hình gtp5g, UPF hoặc networking liên quan.

---

## 1.4.3. Role `worker_join` làm gì?

Đây là module chịu trách nhiệm join worker vào cluster.

### Flow điển hình:

1. Chờ file `/vagrant/ansible/join-command.sh` xuất hiện
2. Chờ master mở port 6443
3. Pause một chút để control-plane ổn định
4. Kiểm tra worker đã join chưa
5. Nếu chưa:
   - `kubeadm reset -f`
   - xóa trạng thái cũ
   - restart runtime/service cần thiết
   - chạy lệnh join
6. Chờ node lên trạng thái `Ready`

---

## 1.4.4. Output/log thường thấy của `worker_join`

```text
PLAY [Join Kubernetes workers] *************************************************

TASK [Gathering Facts] *********************************************************
ok: [k8s-worker1]

TASK [worker_join : Wait for join command file] ********************************
ok: [k8s-worker1]

TASK [worker_join : Wait for kube-apiserver port on master] ********************
ok: [k8s-worker1]

TASK [worker_join : Pause a bit for control-plane stabilization] ***************
Pausing for 20 seconds
ok: [k8s-worker1]

TASK [worker_join : Check if worker already joined] ****************************
ok: [k8s-worker1]

TASK [worker_join : Ensure clean state before join] ****************************
changed: [k8s-worker1]

TASK [worker_join : Join cluster] **********************************************
changed: [k8s-worker1]

TASK [worker_join : Wait until node becomes Ready] *****************************
ok: [k8s-worker1]
```

---

## 1.4.5. Ý nghĩa output của `worker_join`

### `Wait for join command file`
Master phải sinh file join trước, worker mới join được.

### `Wait for kube-apiserver port on master`
Đảm bảo `192.168.56.10:6443` đang mở.

### `Ensure clean state before join`
Dọn tàn dư join cũ để tránh join lỗi.

### `Join cluster`
Nếu task này thành công, worker đã được thêm vào cluster.

### `Wait until node becomes Ready`
Xác nhận Kubernetes nhìn thấy node worker và node đã sẵn sàng.

---

## 1.5. Khi chạy `vagrant up k8s-worker2`

Flow của worker2 gần giống worker1:
- dựng máy
- chạy `common`
- chạy `worker_join`
- có thể chạy `upf`

Nhưng còn một điểm đặc biệt: sau khi worker2 lên xong, dự án có thể kích hoạt **phase 2 auto deploy**.

---

## 1.5.1. Trigger sau khi worker2 up xong

Trong dự án, sau khi `k8s-worker2` lên xong, Vagrant có thể gọi script kiểu:

```bat
scripts\phase2-auto.bat
```

Mục tiêu của script này là tự động chạy tiếp phần:
1. build image
2. load image
3. apply manifest

---

## 1.5.2. Output host thường thấy khi trigger phase 2

```text
[PHASE2] Build images on master...
[PHASE2] Load images into worker1...
[PHASE2] Load images into worker2...
[PHASE2] Apply manifests on master...
[PHASE2 DONE]
```

Nghĩa là sau khi cluster xong, hệ thống tự bước sang giai đoạn deploy core.

---

# 2) Flow phase 2 thật sự là gì?

Phase 2 là giai đoạn triển khai phần 5G core thực tế sau khi cluster đã sẵn sàng.

Nó không còn chỉ là dựng cluster nữa, mà là:
- build image NF
- nạp image vào các node
- apply manifest
- patch placement
- expose service gateway

---

## 2.1. Tổng quan 4 bước của phase 2

Script phase 2 thường chạy 4 bước theo đúng thứ tự:

```bat
[PHASE2] Build images on master...
vagrant provision k8s-master --provision-with phase2-build

[PHASE2] Load images into worker1...
vagrant provision k8s-worker1 --provision-with phase2-load

[PHASE2] Load images into worker2...
vagrant provision k8s-worker2 --provision-with phase2-load

[PHASE2] Apply manifests on master...
vagrant provision k8s-master --provision-with phase2-apply
```

---

## 2.2. Bước 1: `phase2-build` trên master

Lệnh:

```bash
vagrant provision k8s-master --provision-with phase2-build
```

---

## 2.2.1. Module nào chạy?

Provisioner này sẽ gọi playbook:
- `deploy-phase2-build.yml`

Playbook này gọi role:
- `deploy_core_build`

---

## 2.2.2. Role `deploy_core_build` làm gì?

Đây là module build artifact/image.

Flow điển hình:

1. Tạo thư mục `/vagrant/artifacts`
2. Đảm bảo `rsync` đã cài
3. Kiểm tra `docker/nf.Dockerfile` tồn tại
4. Tạo local build root, ví dụ:
   - `/home/vagrant/etri-build`
5. Xóa thư mục build cũ nếu có
6. Copy source từ shared folder sang local disk bằng `rsync`
7. Tạo build context đúng cho repo
8. Build các image NF
9. Save từng image thành `.tar`

---

## 2.2.3. Vì sao phải copy source sang local disk?

Shared folder `/vagrant` đôi khi chậm hoặc có vấn đề với symlink/quyền file.

Nên role build thường:
- copy source sang `/home/vagrant/etri-build`
- build trên local disk trong VM

Việc này giúp ổn định hơn khi `docker build`.

---

## 2.2.4. Output/log thường thấy của `deploy_core_build`

```text
PLAY [Build custom images on master] *******************************************

TASK [Gathering Facts] *********************************************************
ok: [k8s-master]

TASK [deploy_core_build : Ensure artifact directory exists] ********************
ok: [k8s-master]

TASK [deploy_core_build : Ensure rsync is installed] ***************************
ok: [k8s-master]

TASK [deploy_core_build : Check nf Dockerfile exists] **************************
ok: [k8s-master]

TASK [deploy_core_build : Define local build root] *****************************
ok: [k8s-master]

TASK [deploy_core_build : Remove old local build root] *************************
changed: [k8s-master]

TASK [deploy_core_build : Recreate local build root] ***************************
changed: [k8s-master]

TASK [deploy_core_build : Copy source from shared folder to local build root with rsync] ***
changed: [k8s-master]

TASK [deploy_core_build : Create correct b5gc build context] *******************
changed: [k8s-master]

TASK [deploy_core_build : Build required NF images] ****************************
changed: [k8s-master] => (item={'module': 'controller', 'image': 'b5gc-controller:latest'})
changed: [k8s-master] => (item={'module': 'gateway', 'image': 'b5gc-gateway:latest'})
changed: [k8s-master] => (item={'module': 'nsm', 'image': 'b5gc-nsm:latest'})
changed: [k8s-master] => (item={'module': 'pran', 'image': 'b5gc-pran:latest'})
changed: [k8s-master] => (item={'module': 'udm', 'image': 'b5gc-udm:latest'})
changed: [k8s-master] => (item={'module': 'ausf', 'image': 'b5gc-ausf:latest'})
changed: [k8s-master] => (item={'module': 'pcf', 'image': 'b5gc-pcf:latest'})
changed: [k8s-master] => (item={'module': 'damf', 'image': 'b5gc-damf:latest'})
changed: [k8s-master] => (item={'module': 'smf', 'image': 'b5gc-smf:latest'})
changed: [k8s-master] => (item={'module': 'amf', 'image': 'b5gc-amf:latest'})

TASK [deploy_core_build : Save required NF images to tar] **********************
changed: [k8s-master] => (item={'image': 'b5gc-controller:latest', 'tar': 'b5gc-controller.tar'})
changed: [k8s-master] => (item={'image': 'b5gc-gateway:latest', 'tar': 'b5gc-gateway.tar'})
changed: [k8s-master] => (item={'image': 'b5gc-nsm:latest', 'tar': 'b5gc-nsm.tar'})
...
```

---

## 2.2.5. Giải thích module build và output

### `Check nf Dockerfile exists`
Đảm bảo Dockerfile build NF có thật. Nếu thiếu, build sẽ fail ngay.

### `Remove old local build root`
Xóa môi trường build cũ để tránh rác hoặc file cũ gây lỗi.

### `Copy source ... with rsync`
Đồng bộ source sang local build root.

### `Build required NF images`
Đây là khối nặng nhất. Mỗi item tương ứng một module NF:
- controller
- gateway
- nsm
- pran
- udm
- ausf
- pcf
- damf
- smf
- amf

### `Save required NF images to tar`
Xuất image Docker thành `.tar` để các node worker import vào `containerd`.

---

## 2.2.6. Kết quả đầu ra mong đợi sau bước build

Thư mục `/vagrant/artifacts` sẽ có các file như:
- `b5gc-controller.tar`
- `b5gc-gateway.tar`
- `b5gc-nsm.tar`
- `b5gc-pran.tar`
- `b5gc-udm.tar`
- `b5gc-ausf.tar`
- `b5gc-pcf.tar`
- `b5gc-damf.tar`
- `b5gc-smf.tar`
- `b5gc-amf.tar`

Đây là đầu ra rất quan trọng của phase build.

---

## 2.3. Bước 2: `phase2-load` trên worker1

Lệnh:

```bash
vagrant provision k8s-worker1 --provision-with phase2-load
```

---

## 2.3.1. Module nào chạy?

Provisioner này gọi playbook:
- `deploy-phase2-load.yml`

Playbook này gọi role:
- `load_custom_images`

---

## 2.3.2. Role `load_custom_images` làm gì?

Đây là module nạp image từ file `.tar` vào `containerd` của node.

Flow:

1. Kiểm tra công cụ `ctr`
2. Tìm các file `*.tar` trong `/vagrant/artifacts`
3. Fail nếu không có file tar
4. Import từng file tar vào `containerd`
5. In danh sách image đã import

---

## 2.3.3. Vì sao phải load image vào worker?

Manifest của bạn dùng `imagePullPolicy: Never`.

Điều đó có nghĩa:
- Kubernetes sẽ **không pull image từ registry**
- pod chỉ chạy được nếu image đã có sẵn ngay trên node nơi pod được schedule tới

Vì vậy:
- build trên master là chưa đủ
- worker1 và worker2 đều phải có image đúng

---

## 2.3.4. Output/log thường thấy của `load_custom_images`

```text
PLAY [Load custom images into node containerd] *********************************

TASK [Gathering Facts] *********************************************************
ok: [k8s-worker1]

TASK [load_custom_images : Ensure ctr is available] ****************************
ok: [k8s-worker1]

TASK [load_custom_images : Find image tar files] *******************************
ok: [k8s-worker1]

TASK [load_custom_images : Fail if no image tar files found] *******************
skipping: [k8s-worker1]

TASK [load_custom_images : Import image tar files into containerd] *************
changed: [k8s-worker1] => (item=/vagrant/artifacts/b5gc-controller.tar)
changed: [k8s-worker1] => (item=/vagrant/artifacts/b5gc-gateway.tar)
changed: [k8s-worker1] => (item=/vagrant/artifacts/b5gc-nsm.tar)
changed: [k8s-worker1] => (item=/vagrant/artifacts/b5gc-pran.tar)
...

TASK [load_custom_images : Verify imported images] *****************************
ok: [k8s-worker1]

TASK [load_custom_images : Print imported images] ******************************
ok: [k8s-worker1] => {
  "imported_images.stdout_lines": [
    "REF TYPE DIGEST SIZE PLATFORMS LABELS",
    ...
  ]
}
```

---

## 2.3.5. Giải thích output

### `Find image tar files`
Role quét thư mục artifact để xem có tar nào không.

### `Fail if no image tar files found`
Nếu không có file `.tar`, bước này sẽ fail vì không có gì để import.

### `Import image tar files into containerd`
Đây là bước thật sự nạp image vào node bằng lệnh kiểu:

```bash
ctr -n k8s.io images import /vagrant/artifacts/b5gc-controller.tar
```

### `Verify imported images`
Kiểm tra image đã tồn tại trong namespace `k8s.io` của containerd chưa.

---

## 2.3.6. Kết quả mong đợi sau bước này

Node `k8s-worker1` đã có image local trong `containerd`.  
Những pod được schedule vào worker1 sẽ có thể start mà không cần pull registry.

---

## 2.4. Bước 3: `phase2-load` trên worker2

Lệnh:

```bash
vagrant provision k8s-worker2 --provision-with phase2-load
```

Flow hoàn toàn tương tự worker1, chỉ khác là image được nạp vào worker2.

Điều này đặc biệt quan trọng cho các pod edge như:
- `pran`
- các `amf` đặt ở `site=edge`

Nếu worker2 không có image, các pod edge sẽ lỗi khi khởi động.

---

## 2.5. Bước 4: `phase2-apply` trên master

Lệnh:

```bash
vagrant provision k8s-master --provision-with phase2-apply
```

Đây là bước triển khai thực tế toàn bộ 5G core lên cluster.

---

## 2.5.1. Các module/role chạy trong `phase2-apply`

Thường gồm 3 role chính:

1. `label_nodes`
2. `deploy_core`
3. `expose_gateway`

---

## 2.5.2. Role `label_nodes` làm gì?

Đây là module gắn nhãn node để Kubernetes biết node nào là:
- control
- centre
- edge

Ví dụ:
- `k8s-master` → control
- `k8s-worker1` → `site=centre`
- `k8s-worker2` → `site=edge`

### Output thường thấy

```text
TASK [label_nodes : Wait until k8s-master exists] ******************************
ok: [k8s-master]

TASK [label_nodes : Wait until k8s-worker1 exists] *****************************
ok: [k8s-master]

TASK [label_nodes : Wait until k8s-worker2 exists] *****************************
ok: [k8s-master]

TASK [label_nodes : Label master as control role] ******************************
ok: [k8s-master]

TASK [label_nodes : Label worker1 as centre role] ******************************
ok: [k8s-master]

TASK [label_nodes : Label worker2 as edge role] ********************************
ok: [k8s-master]

TASK [label_nodes : Label worker1 site=centre] *********************************
ok: [k8s-master]

TASK [label_nodes : Label worker2 site=edge] ***********************************
ok: [k8s-master]
```

### Ý nghĩa
Các nhãn này là cơ sở để `nodeSelector` quyết định pod chạy trên node nào.

---

## 2.5.3. Role `deploy_core` làm gì?

Đây là module lớn nhất và quan trọng nhất của phase 2.

Nó chịu trách nhiệm:
- render manifest
- apply namespace
- deploy controller/gateway/nsm/các NF
- chờ rollout
- patch lại replicas và nodeSelector
- in trạng thái cuối

---

## 2.5.4. Giai đoạn render manifest

Role sẽ sinh các file như:
- `namespace.yaml`
- `controller.yaml`
- `gateway.yaml`
- `nsm.yaml`
- `pran.yaml`

vào thư mục tạm, ví dụ:
- `/tmp/etri-generated`

### Output thường thấy

```text
TASK [deploy_core : Ensure generated manifest directory exists] ****************
changed: [k8s-master]

TASK [deploy_core : Render namespace manifest] *********************************
changed: [k8s-master]

TASK [deploy_core : Render custom controller manifest] *************************
changed: [k8s-master]

TASK [deploy_core : Render custom gateway manifest] ****************************
changed: [k8s-master]

TASK [deploy_core : Render custom NSM manifest] ********************************
changed: [k8s-master]

TASK [deploy_core : Render PRAN manifest] **************************************
changed: [k8s-master]
```

### Ý nghĩa
Đây là bước chuyển từ template `.j2` + biến Ansible thành file manifest YAML thật để `kubectl apply`.

---

## 2.5.5. Apply namespace và controller

### Output thường thấy

```text
TASK [deploy_core : Apply namespace] *******************************************
ok: [k8s-master]

TASK [deploy_core : Apply controller] ******************************************
changed: [k8s-master]

TASK [deploy_core : Wait for controller rollout] *******************************
ok: [k8s-master]
```

### Ý nghĩa
- Namespace tạo vùng làm việc riêng cho 5GC
- Controller là thành phần điều phối đầu tiên cần lên

Nếu `Wait for controller rollout` timeout, nghĩa là Deployment đã apply nhưng pod controller chưa Available.

---

## 2.5.6. Apply gateway

### Output thường thấy

```text
TASK [deploy_core : Apply gateway] *********************************************
changed: [k8s-master]

TASK [deploy_core : Wait for gateway rollout] **********************************
ok: [k8s-master]

TASK [deploy_core : Wait for gateway service endpoints] ************************
ok: [k8s-master]
```

### Ý nghĩa
Gateway phải có pod chạy và service phải sinh endpoint. Nếu endpoint rỗng thì có thể:
- pod chưa ready
- selector không match
- container crash

---

## 2.5.7. Apply NSM

Đây là block rất quan trọng vì NSM là thành phần bạn từng hay lỗi rollout.

### Output thường thấy

```text
TASK [deploy_core : Check whether NSM deployment exists] ***********************
ok: [k8s-master]

TASK [deploy_core : Scale down old NSM deployment to zero] *********************
changed: [k8s-master]

TASK [deploy_core : Force delete old NSM pods if exist] ************************
ok: [k8s-master]

TASK [deploy_core : Wait until all old NSM pods are gone] **********************
ok: [k8s-master]

TASK [deploy_core : Apply NSM] *************************************************
changed: [k8s-master]

TASK [deploy_core : Wait for NSM rollout] **************************************
ok: [k8s-master]
```

### Ý nghĩa chi tiết

- `Check whether NSM deployment exists`: xem có deployment cũ không
- `Scale down old NSM deployment to zero`: hạ replica về 0 để dọn pod cũ
- `Force delete old NSM pods if exist`: xóa sạch pod cũ nếu cần
- `Apply NSM`: apply manifest mới
- `Wait for NSM rollout`: chờ deployment đạt Available

Nếu xuất hiện lỗi:

```text
error: timed out waiting for the condition
Waiting for deployment "nsm-dep" rollout to finish: 0 of 1 updated replicas are available...
```

thì nghĩa là:
- deployment đã được tạo
- nhưng pod NSM không chạy healthy
- thường do image/config/readiness/liveness/nodeSelector

---

## 2.5.8. Apply các NF còn lại

Sau NSM, hệ thống thường apply tiếp:
- UDR
- UDM
- AUSF
- PCF
- DAMF
- SMF
- AMF
- PRAN

### Output mẫu

```text
TASK [deploy_core : Apply UDR] *************************************************
changed: [k8s-master]

TASK [deploy_core : Wait for UDR rollout] **************************************
ok: [k8s-master]

TASK [deploy_core : Apply UDM] *************************************************
changed: [k8s-master]

TASK [deploy_core : Apply AUSF] ************************************************
changed: [k8s-master]

TASK [deploy_core : Apply PCF] *************************************************
changed: [k8s-master]

TASK [deploy_core : Wait for UDM rollout] **************************************
ok: [k8s-master]

TASK [deploy_core : Wait for AUSF rollout] *************************************
ok: [k8s-master]

TASK [deploy_core : Wait for PCF rollout] **************************************
ok: [k8s-master]

TASK [deploy_core : Apply DAMF] ************************************************
changed: [k8s-master]

TASK [deploy_core : Apply SMF] *************************************************
changed: [k8s-master]

TASK [deploy_core : Apply AMF 10-100] *****************************************
changed: [k8s-master]

TASK [deploy_core : Apply AMF 10-101] *****************************************
changed: [k8s-master]

TASK [deploy_core : Apply AMF 10-102] *****************************************
changed: [k8s-master]

TASK [deploy_core : Apply PRAN] ************************************************
changed: [k8s-master]
```

### Ý nghĩa
Đây là giai đoạn toàn bộ lõi 5G được đưa lên cluster.

---

## 2.5.9. Patch replicas và nodeSelector

Đây là phần cực kỳ quan trọng để ép pod chạy đúng node.

Ví dụ:
- NF thuộc centre chạy trên `k8s-worker1`
- NF thuộc edge chạy trên `k8s-worker2`

### Output mẫu

```text
TASK [deploy_core : Patch replicas and nodeSelector for deployments] ***********
changed: [k8s-master] => (item={'key': 'controller', 'value': {'deployment': 'ctrl-dep', 'replicas': 1, 'site': 'centre'}})
changed: [k8s-master] => (item={'key': 'gateway', 'value': {'deployment': 'gateway-dep', 'replicas': 1, 'site': 'centre'}})
changed: [k8s-master] => (item={'key': 'nsm', 'value': {'deployment': 'nsm-dep', 'replicas': 1, 'site': 'centre'}})
...
changed: [k8s-master] => (item={'key': 'amf_10_100', 'value': {'deployment': 'amf-10-100-dep', 'replicas': 2, 'site': 'edge'}})
...
changed: [k8s-master] => (item={'key': 'pran', 'value': {'deployment': 'pran-dep', 'replicas': 1, 'site': 'edge'}})
```

### Ý nghĩa
Dù manifest gốc chưa đúng placement, bước patch sẽ sửa lại.

Đây là lý do dự án của bạn có thể kiểm soát:
- node nào chạy control NF
- node nào chạy edge NF

---

## 2.5.10. In trạng thái cuối cùng của pod và service

### Output mẫu

```text
TASK [deploy_core : Show core pods] ********************************************
ok: [k8s-master]

TASK [deploy_core : Print core pod list] ***************************************
ok: [k8s-master]

TASK [deploy_core : Show core services] ****************************************
ok: [k8s-master]

TASK [deploy_core : Print core service list] ***********************************
ok: [k8s-master]
```

Thông thường phần output thực tế sẽ có danh sách như:

```text
NAME                               READY   STATUS             RESTARTS   AGE   IP            NODE
ctrl-dep-xxxxx                     1/1     Running            0          ...   10.244.x.x    k8s-worker1
gateway-dep-xxxxx                  1/1     Running            0          ...   10.244.x.x    k8s-worker1
nsm-dep-xxxxx                      0/1     CrashLoopBackOff   5          ...   10.244.x.x    k8s-worker1
pran-dep-xxxxx                     1/1     Running            0          ...   10.244.x.x    k8s-worker2
amf-10-100-dep-xxxxx               0/1     CrashLoopBackOff   7          ...   10.244.x.x    k8s-worker2
```

### Ý nghĩa
Đây là block quan trọng nhất để đánh giá deploy thành công hay chưa.

Những cột bạn cần đọc:
- `READY`
- `STATUS`
- `RESTARTS`
- `NODE`

---

## 2.5.11. Role `expose_gateway` làm gì?

Sau khi core được deploy, role này sẽ patch service gateway để có thể truy cập từ bên ngoài qua NodePort.

### Output mẫu

```text
TASK [expose_gateway : Wait for gateway deployment] ****************************
ok: [k8s-master]

TASK [expose_gateway : Wait for gateway service] *******************************
ok: [k8s-master]

TASK [expose_gateway : Ensure gateway service is NodePort 30007] **************
changed: [k8s-master]

TASK [expose_gateway : Show gateway service] ***********************************
ok: [k8s-master]

TASK [expose_gateway : Print gateway service] **********************************
ok: [k8s-master]
```

### Ý nghĩa
Service gateway được đổi sang `NodePort`, ví dụ `30007`, để bạn có thể truy cập qua IP node + port đó.

---

# Kết luận

Mục 1 và 2 của dự án mô tả toàn bộ flow vận hành thực tế như sau:

1. `vagrant up k8s-master`
   - dựng master
   - chạy `common`
   - chạy `master_init`
   - sinh cluster và join command

2. `vagrant up k8s-worker1`
   - dựng worker1
   - chạy `common`
   - chạy `worker_join`

3. `vagrant up k8s-worker2`
   - dựng worker2
   - chạy `common`
   - chạy `worker_join`
   - có thể tự trigger phase 2

4. `phase2-build`
   - role `deploy_core_build`
   - build image NF
   - xuất `.tar`

5. `phase2-load`
   - role `load_custom_images`
   - import `.tar` vào containerd trên worker

6. `phase2-apply`
   - role `label_nodes`
   - role `deploy_core`
   - role `expose_gateway`

Toàn bộ các module này ghép lại thành pipeline hoàn chỉnh:
- dựng hạ tầng
- tạo cluster
- build artifact
- nạp image
- deploy 5GC
- ép placement theo centre/edge
- expose gateway ra ngoài
