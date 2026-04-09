# Flow deploy 5GC bằng Vagrant + Ansible + Kubernetes

## 1. Mục tiêu của flow này
Flow này dùng để dựng cụm máy ảo bằng Vagrant, cài Kubernetes bằng Ansible, sau đó build image NF, load image sang các node worker, rồi apply manifest để chạy hệ 5GC trên cluster. Ý nghĩa của việc tách flow như vậy là để dễ debug: phase hạ tầng tách riêng khỏi phase deploy ứng dụng; build tách khỏi load; load tách khỏi apply. Khi có lỗi, có thể chạy lại đúng node hoặc đúng phase mà không phải làm lại toàn bộ từ đầu.

## 2. Tổng quan thứ tự chạy đúng
Thứ tự chạy chuẩn của bạn là:
1. `vagrant up k8s-master`
2. `vagrant up k8s-worker1`
3. `vagrant up k8s-worker2`
4. Nếu muốn chạy lại Ansible cho từng node hạ tầng:
   - `vagrant provision k8s-master`
   - `vagrant provision k8s-worker1`
   - `vagrant provision k8s-worker2`
5. Qua phase deploy:
   - `vagrant provision k8s-master --provision-with phase2-build`
   - `vagrant provision k8s-worker1 --provision-with phase2-load`
   - `vagrant provision k8s-worker2 --provision-with phase2-load`
   - `vagrant provision k8s-master --provision-with phase2-apply`

Đây là flow rất hợp lý vì:
- master build image
- worker1 và worker2 load image vào container runtime của node
- master apply manifest để Kubernetes tạo Pod trên các node phù hợp

## 3. Giai đoạn 1: `vagrant up` từng node có ý nghĩa gì
### 3.1 `vagrant up k8s-master`
Lệnh này tạo và khởi động VM `k8s-master`, mount thư mục project từ máy host vào VM, cấu hình mạng, rồi chạy provisioner mặc định gắn với node đó.

Ví dụ:
```bash
vagrant up k8s-master
```

### Output thường gặp và ý nghĩa
- `Bringing machine 'k8s-master' up with 'virtualbox' provider...`
  - Vagrant đang gọi VirtualBox để dựng máy ảo master.
- `Importing base box ...`
  - Đang lấy box Ubuntu nền.
- `Matching MAC address for NAT networking...`
  - Đang gắn card NAT.
- `Setting the name of the VM...`
  - Đặt tên máy ảo trong VirtualBox.
- `Clearing any previously set network interfaces...`
  - Xóa cấu hình mạng cũ để cấu hình lại.
- `Preparing network interfaces based on configuration...`
  - Chuẩn bị card mạng như NAT và host-only.
- `Forwarding ports...`
  - Nếu có port forwarding thì Vagrant sẽ khai báo ở bước này.
- `Booting VM...`
  - Máy ảo đang bật lên.
- `Waiting for machine to boot. This may take a few minutes...`
  - Đang chờ SSH sẵn sàng.
- `Machine booted and ready!`
  - VM đã lên hoàn chỉnh.
- `Mounting shared folders...`
  - Thư mục project trên Windows được mount vào VM, thường là `/vagrant`.
- `Running provisioner: ansible_local...`
  - Bắt đầu chạy Ansible ngay trong VM.
- `ok: [...]`
  - Task chạy thành công, không đổi gì.
- `changed: [...]`
  - Task chạy thành công và có thay đổi hệ thống.
- `fatal: [...] FAILED!`
  - Provision lỗi tại task đó.

### Ý nghĩa thực tế
Sau lệnh này, master thường sẽ có:
- hệ điều hành đã lên
- containerd / kubelet / kubeadm được cài
- node control-plane được init
- kubectl có thể dùng trên master
- có thể sinh join command cho worker

---

### 3.2 `vagrant up k8s-worker1`
```bash
vagrant up k8s-worker1
```
Ý nghĩa tương tự nhưng dành cho worker1. Worker này thường được join vào cluster sau khi master sẵn sàng.

### Output thường gặp
Các dòng đầu giống master, nhưng phần Ansible thường mang ý nghĩa:
- cài runtime
- cài kubelet, kubeadm
- nhận join command
- join vào cluster
- gắn label như `site=centre` nếu playbook có cấu hình

### Kết quả mong đợi
Worker1 xuất hiện trong:
```bash
kubectl get nodes -o wide
```
với trạng thái `Ready` hoặc tối thiểu đã join cluster.

---

### 3.3 `vagrant up k8s-worker2`
```bash
vagrant up k8s-worker2
```
Tương tự worker1, nhưng thường được label `site=edge` hoặc vai trò khác tùy playbook.

### Kết quả mong đợi
Sau khi xong cả 3 node:
```bash
kubectl get nodes -o wide
```
sẽ có dạng gần giống:
```bash
NAME          STATUS   ROLES                  AGE   VERSION    INTERNAL-IP
k8s-master    Ready    control-plane,master   ...   v1.29.15   192.168.56.10
k8s-worker1   Ready    <none>                 ...   v1.29.15   192.168.56.11
k8s-worker2   Ready    <none>                 ...   v1.29.15   192.168.56.12
```

### Ý nghĩa từng cột
- `NAME`: tên node
- `STATUS`: trạng thái node; `Ready` là dùng được, `NotReady` là có lỗi kubelet/CNI/network
- `ROLES`: control-plane hay worker
- `VERSION`: bản Kubernetes
- `INTERNAL-IP`: IP nội bộ Kubernetes dùng để giao tiếp giữa các node

## 4. Nếu muốn chạy lại Ansible cho từng node hạ tầng
Sau khi `vagrant up`, nếu muốn chạy lại phần provision mặc định cho từng node thì dùng:

```bash
vagrant provision k8s-master
vagrant provision k8s-worker1
vagrant provision k8s-worker2
```

### Ý nghĩa
Các lệnh này không tạo lại VM, chỉ chạy lại provisioner đã khai báo trong Vagrantfile cho từng node. Nó rất hữu ích khi:
- bạn sửa role Ansible
- muốn cài lại package
- muốn sửa kubelet config
- muốn chạy lại join logic
- không muốn destroy/recreate máy ảo

### Output thường gặp
- `Running provisioner: ansible_local...`
  - Vagrant đang gọi lại Ansible Local.
- `PLAY [all]`
  - Bắt đầu play.
- `TASK [role_name : task_name]`
  - Đang chạy task cụ thể trong role.
- `ok`
  - Task không cần thay đổi gì.
- `changed`
  - Task có thay đổi.
- `skipping`
  - Task bị bỏ qua vì điều kiện `when`.
- `failed`
  - Task lỗi.

### Ý nghĩa theo từng node
#### `vagrant provision k8s-master`
Thường chạy lại các module:
- `common`
- `master_init`
- các handler restart kubelet/containerd
- copy kubeconfig, enable kubectl, setup control-plane

#### `vagrant provision k8s-worker1`
Thường chạy lại:
- `common`
- `worker_join`
- fix kubelet args
- join lại cluster nếu cần
- gán label/taint tùy thiết kế

#### `vagrant provision k8s-worker2`
Giống worker1 nhưng áp dụng trên worker2.

## 5. Giai đoạn deploy phase 2: đúng thứ tự lệnh
Sau khi hạ tầng OK, sang phase deploy thì chạy đúng các lệnh:

```bash
vagrant provision k8s-master --provision-with phase2-build
vagrant provision k8s-worker1 --provision-with phase2-load
vagrant provision k8s-worker2 --provision-with phase2-load
vagrant provision k8s-master --provision-with phase2-apply
```

Đây là phần quan trọng nhất của flow deploy ứng dụng.

## 6. `phase2-build` trên master làm gì
Lệnh:
```bash
vagrant provision k8s-master --provision-with phase2-build
```

### Mục đích
Dùng master làm nơi build Docker image cho các network function và thành phần liên quan trước khi deploy.

### Các module/role thường nằm trong phase này
Tùy Vagrantfile và playbook, phase này thường gọi role kiểu:
- `deploy_core_build`
- hoặc `build_push_registry`
- hoặc playbook riêng để build image NF

### Những việc role này thường làm
1. Vào thư mục project trong `/vagrant`
2. Tìm Dockerfile, thường là `docker/nf.Dockerfile`
3. Build từng image với `--build-arg B5GC_MODULE=...`
4. Tạo image như:
   - `b5gc-nsm:latest`
   - `b5gc-gateway:latest`
   - `b5gc-controller:latest`
   - `b5gc-pran:latest`
   - và các NF khác như amf, smf, udm, ausf...
5. Nếu có registry local thì có thể tag thêm:
   - `192.168.56.10:5000/b5gc-nsm:latest`

### Output thường gặp
- `TASK [deploy_core_build : Build required NF images]`
  - Task build image đang chạy.
- `docker build -t b5gc-nsm:latest ...`
  - Lệnh build thực tế.
- `Successfully built <image_id>`
  - Docker build thành công.
- `Successfully tagged b5gc-nsm:latest`
  - Image đã được gắn tag.
- `changed: [k8s-master]`
  - Build xong và có thay đổi.
- `failed: [k8s-master]`
  - Build lỗi; cần xem stderr.

### Ý nghĩa nếu lỗi
- lỗi đường dẫn Dockerfile: role build trỏ sai file
- lỗi source code: module Go/C/C++ build hỏng
- lỗi biến `B5GC_MODULE`: tên module sai
- lỗi image/tag: manifest sau này sẽ không kéo được đúng image

### Cách kiểm tra sau build
Trên master:
```bash
docker images
```
hoặc nếu dùng containerd:
```bash
sudo ctr -n k8s.io images ls
```

### Ý nghĩa output kiểm tra
Nếu thấy các image NF xuất hiện, phase build đã xong.

## 7. `phase2-load` trên worker1 làm gì
Lệnh:
```bash
vagrant provision k8s-worker1 --provision-with phase2-load
```

### Mục đích
Nạp image đã build vào runtime của worker1 để Pod trên worker1 có thể chạy mà không phải pull từ Internet.

### Module/role thường dùng
- `deploy_core_load`
- hoặc task import image tar
- hoặc pull từ local registry / ctr images import / docker save-load

### Những việc role này thường làm
1. Lấy image từ master hoặc từ thư mục chia sẻ
2. Save thành tar hoặc dùng registry local
3. Trên worker import image vào containerd
4. Kiểm tra image đã có trong runtime của worker

### Output thường gặp
- `TASK [load images to worker runtime]`
  - Đang nạp image vào worker.
- `ctr -n k8s.io images import ...`
  - Đang import image vào containerd.
- `Loaded image: b5gc-nsm:latest`
  - Image đã được load xong.
- `changed: [k8s-worker1]`
  - Có thay đổi, tức là image đã được import.
- `ok: [k8s-worker1]`
  - Có thể image đã có sẵn từ trước.

### Ý nghĩa
Worker1 giờ đã có image cục bộ, nên khi Kubernetes schedule Pod lên worker1 với `imagePullPolicy: Never` hoặc `IfNotPresent`, pod vẫn chạy được.

## 8. `phase2-load` trên worker2 làm gì
Lệnh:
```bash
vagrant provision k8s-worker2 --provision-with phase2-load
```

Ý nghĩa giống worker1 nhưng áp dụng cho worker2. Vì scheduler có thể đặt pod lên cả worker1 lẫn worker2, nên cả hai node đều phải có image cần dùng.

### Nếu bỏ qua bước này thì sao
Pod có thể bị:
- `ImagePullBackOff`
- `ErrImageNeverPull`
- hoặc stuck ở `ContainerCreating`

nếu manifest đang yêu cầu image local mà worker chưa có image đó.

## 9. `phase2-apply` trên master làm gì
Lệnh:
```bash
vagrant provision k8s-master --provision-with phase2-apply
```

### Mục đích
Áp manifest Kubernetes hoặc Helm chart để tạo namespace, configmap, service, deployment, rồi chờ rollout.

### Các module/role thường có trong phase này
- `deploy_core`
- `deploy_core_apply`
- `deploy_core_helm`
- hoặc playbook apply namespace, controller, gateway, nsm, pran, các NF khác

### Những việc bên trong thường diễn ra
1. Tạo namespace `etri6g`
2. Tạo configmap cho NF
3. Apply service và deployment
4. Gán `nodeSelector` để pod chạy đúng worker
5. Chạy `kubectl rollout status`
6. Nếu dùng Helm thì render values rồi `helm upgrade --install`

### Output thường gặp
- `TASK [Apply namespace manifest]`
  - Tạo namespace cho hệ thống.
- `namespace/etri6g created`
  - Namespace mới được tạo.
- `namespace/etri6g configured`
  - Namespace đã tồn tại, nay được cập nhật.
- `configmap/nsm-config created`
  - Config cho NSM được tạo.
- `service/gateway-svc created`
  - Service gateway được tạo.
- `deployment.apps/nsm-dep created`
  - Deployment NSM được tạo.
- `deployment.apps/nsm-dep configured`
  - Deployment có sẵn, được update.
- `Waiting for deployment "nsm-dep" rollout to finish`
  - Kubernetes đang chờ pod mới lên và pod cũ hạ xuống.
- `deployment "nsm-dep" successfully rolled out`
  - Rollout thành công.

### Nếu dùng Helm
Có thể thấy:
- `helm lint ...`
  - Kiểm tra chart.
- `Release "etrib5gc" has been upgraded. Happy Helming!`
  - Helm deploy/update thành công.
- `Error: ...`
  - Helm lỗi template, values, ownership namespace, hoặc label Helm.

## 10. Giải thích rõ module trong toàn flow
## 10.1 Module/role hạ tầng
### `common`
Role nền tảng, thường dùng để:
- cài package chung
- cấu hình containerd
- cấu hình sysctl, kernel module
- tắt swap
- chuẩn bị môi trường cho kubeadm

### `master_init`
Role dành cho master:
- `kubeadm init`
- tạo kubeconfig cho user
- cài CNI
- sinh join command cho worker
- cấu hình control-plane node

### `worker_join`
Role dành cho worker:
- nhận join token từ master
- `kubeadm join`
- chỉnh kubelet args
- set node IP
- đảm bảo worker vào cluster

## 10.2 Module/role deploy ứng dụng
### `deploy_core_build`
Build image NF và thành phần cần thiết.

### `deploy_core_load`
Load image từ nguồn build vào runtime của worker.

### `deploy_core`
Role apply manifest từ template Jinja2.

### `deploy_core_apply`
Biến thể apply trực tiếp manifest sau khi render.

### `deploy_core_helm`
Dùng Helm chart để lint, render preview, và deploy release.

## 11. Output của từng lệnh sau deploy nên hiểu thế nào
### 11.1 Kiểm tra node
```bash
kubectl get nodes -o wide
```
#### Output có thể gặp
- `Ready`: node dùng được
- `NotReady`: node lỗi kubelet, CNI, network hoặc container runtime
- `SchedulingDisabled`: node đã cordon, không nhận pod mới

### 11.2 Kiểm tra pod
```bash
kubectl -n etri6g get pods -o wide
```
#### Ý nghĩa các trạng thái
- `Running`: container đang chạy
- `Pending`: chưa schedule được hoặc chờ resource
- `ContainerCreating`: đang mount volume, pull/load image, tạo container
- `CrashLoopBackOff`: container khởi động rồi chết liên tục
- `ImagePullBackOff`: không pull được image
- `ErrImageNeverPull`: manifest bảo không pull nhưng node không có image
- `Completed`: job chạy xong

#### Cột `READY`
- `1/1`: container duy nhất đã ready
- `0/1`: container chưa ready hoặc chết
- `2/2`: pod có 2 container và cả 2 đều ready

### 11.3 Xem rollout
```bash
kubectl -n etri6g rollout status deploy/nsm-dep
```
#### Output
- `deployment "nsm-dep" successfully rolled out`
  - Pod mới đã lên tốt.
- `Waiting for deployment ... 0 of 1 updated replicas are available`
  - Deployment đã tạo pod nhưng pod chưa Ready.
- `error: timed out waiting for the condition`
  - Hết thời gian chờ, thường phải quay lại xem pod/log.

### 11.4 Xem mô tả pod
```bash
kubectl -n etri6g describe pod <pod-name>
```
#### Ý nghĩa
Lệnh này dùng để xem:
- pod đang nằm trên node nào
- image gì
- event lỗi gì
- probe lỗi ra sao
- volume mount có fail không

Các dòng quan trọng:
- `Node:` pod đang chạy trên node nào
- `Image:` image thực tế
- `State:` running/waiting/terminated
- `Last State:` container trước đó chết như thế nào
- `Events:` timeline lỗi/sự kiện

### 11.5 Xem log pod
```bash
kubectl -n etri6g logs <pod-name>
kubectl -n etri6g logs <pod-name> --previous
```
#### Ý nghĩa
- log thường: log lần chạy hiện tại
- `--previous`: log lần crash trước đó, rất quan trọng khi pod đang CrashLoopBackOff

### 11.6 Xem log containerd khi pod chết sớm
```bash
sudo crictl ps -a
sudo crictl logs <container-id>
```
#### Ý nghĩa
Hữu ích khi container restart quá nhanh, kubectl logs không đủ thông tin.

## 12. Flow debug đúng thứ tự sau khi chạy xong phase2-apply
Sau khi chạy:
```bash
vagrant provision k8s-master --provision-with phase2-apply
```
nếu có pod lỗi thì debug theo thứ tự này:

```bash
kubectl get nodes -o wide
kubectl -n etri6g get pods -o wide
kubectl -n etri6g describe pod <pod-name>
kubectl -n etri6g logs <pod-name> --previous
kubectl -n etri6g get svc
kubectl -n etri6g get endpoints
kubectl -n etri6g get cm
```

### Ý nghĩa
- kiểm tra node trước
- rồi kiểm tra pod
- rồi mới vào log
- rồi kiểm tra service, endpoint, configmap
- như vậy sẽ khoanh vùng nhanh hơn

## 13. Ý nghĩa tổng thể của từng lệnh trong flow của bạn
### `vagrant up k8s-master`
Dựng master từ đầu, cài control-plane và chuẩn bị cluster.

### `vagrant up k8s-worker1`
Dựng worker1 từ đầu, join cluster, sẵn sàng chạy pod.

### `vagrant up k8s-worker2`
Dựng worker2 từ đầu, join cluster, sẵn sàng chạy pod.

### `vagrant provision k8s-master`
Chạy lại playbook hạ tầng của master mà không cần tạo lại VM.

### `vagrant provision k8s-worker1`
Chạy lại playbook hạ tầng của worker1.

### `vagrant provision k8s-worker2`
Chạy lại playbook hạ tầng của worker2.

### `vagrant provision k8s-master --provision-with phase2-build`
Build toàn bộ image NF ở master.

### `vagrant provision k8s-worker1 --provision-with phase2-load`
Load image vào worker1 để worker1 chạy được pod local image.

### `vagrant provision k8s-worker2 --provision-with phase2-load`
Load image vào worker2 để worker2 chạy được pod local image.

### `vagrant provision k8s-master --provision-with phase2-apply`
Apply manifest/Helm để tạo namespace, config, service, deployment và rollout pod.

## 14. Kết luận ngắn gọn
Flow chuẩn của bạn là:
- dựng từng node bằng `vagrant up`
- cần chạy lại hạ tầng node nào thì `vagrant provision <node>`
- build image trên master bằng `phase2-build`
- load image sang từng worker bằng `phase2-load`
- apply toàn bộ tài nguyên Kubernetes từ master bằng `phase2-apply`

Chính nhờ cách chia phase này mà khi lỗi bạn biết ngay lỗi nằm ở:
- hạ tầng node
- build image
- load image
- hay apply/rollout Kubernetes
