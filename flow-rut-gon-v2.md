# Flow Vagrant + Ansible (Rút gọn + giải thích module)

---

# 1) Flow khi chạy Vagrant

## 1.1 `vagrant up k8s-master`
**Vagrant làm:**
- Tạo VM, gắn mạng (NAT + host-only)
- Mount project vào `/vagrant`
- Chạy `ansible_local`

**Ý nghĩa log chính:**
- `Booting/Mounting` → VM sẵn sàng, có source
- `ansible_local` → bắt đầu cấu hình hệ thống

---

## 1.2 Ansible trên master

### Module/Role: `common` (nền hệ thống)
**Mục tiêu:** biến Ubuntu → node chạy được Kubernetes
- Tắt swap (K8s yêu cầu)
- Kernel modules (`overlay`, `br_netfilter`)
- `sysctl` (network cho pod)
- Cài runtime: `containerd` (+ docker để build)
- Cài `kubeadm`, `kubelet`, `kubectl`
- Bật `kubelet`

**Log hiểu nhanh:**
- `ok` → đã đúng trạng thái
- `changed` → vừa cấu hình/cài đặt xong

---

### Module/Role: `master_init` (khởi tạo cluster)
**Mục tiêu:** tạo control-plane
- `kubeadm init`
- Cài CNI (Flannel)
- Tạo kubeconfig cho user
- Sinh `join-command.sh` cho worker

**Log quan trọng:**
- `Init cluster → changed` → cluster vừa tạo
- `Apply Flannel` → network pod hoạt động
- `Create join command` → sẵn sàng cho worker join

---

## 1.3 `vagrant up k8s-worker1/2`

### Module/Role: `worker_join`
**Mục tiêu:** thêm worker vào cluster
- Đọc `join-command.sh`
- `kubeadm join`
- Chờ node `Ready`

**Log:**
- `Join cluster → changed` → join thành công
- `Wait Ready → ok` → node usable

> Worker2 có thể trigger Phase2 (deploy core)

---

# 2) Flow Phase 2 (Deploy 5GC)

## 2.1 Tổng pipeline
```
build (master) → load (workers) → apply (master)
```

---

## 2.2 Module: `deploy_core_build` (BUILD)
**Mục tiêu:** tạo image NF
- Copy source sang local (tránh lỗi `/vagrant`)
- Build các NF: controller, gateway, nsm, pran, udm, ausf, pcf, damf, smf, amf
- `docker save` → xuất `.tar`

**Output:**
```
/vagrant/artifacts/*.tar
```

**Ý nghĩa:**
- Tách build khỏi runtime, không cần registry

---

## 2.3 Module: `load_custom_images` (LOAD)
**Mục tiêu:** nạp image vào node chạy pod
- Tìm `*.tar`
- `ctr -n k8s.io images import ...`

**Vì sao cần:**
- Manifest dùng `imagePullPolicy: Never`
→ Node PHẢI có sẵn image

---

## 2.4 Phase APPLY (master)

### Module: `label_nodes`
**Mục tiêu:** điều khiển placement
- `site=centre` → worker1
- `site=edge` → worker2

→ Là cơ sở cho `nodeSelector`

---

### Module: `deploy_core` (DEPLOY)
**Mục tiêu:** đưa toàn bộ 5GC lên K8s

**Flow rút gọn:**
1. Render template → YAML
2. Apply namespace
3. Deploy controller, gateway
4. Deploy NSM
5. Deploy NF còn lại
6. Patch `nodeSelector` + `replicas`
7. In trạng thái

**Giải thích nhanh các bước:**

- **Render**: chuyển `.j2` + biến → YAML thật
- **Apply**: `kubectl apply`
- **Rollout**: chờ pod đạt `Available`
- **Patch**: ép pod chạy đúng node (centre/edge)

---

### Lỗi thường gặp (đọc log cực nhanh)

| Log | Ý nghĩa | Nguyên nhân |
|-----|--------|------------|
| `0/1 replicas available` | pod chưa ready | thiếu image / crash |
| `CrashLoopBackOff` | container chết lặp | config sai / service chưa sẵn |
| `ImagePullBackOff` | không có image | chưa load vào node |

---

## 2.5 Module: `expose_gateway`
**Mục tiêu:** mở truy cập từ ngoài
- Patch service → `NodePort`

→ Truy cập qua:
```
<NodeIP>:<NodePort>
```

---

# Kết luận (cốt lõi cần nhớ)

## Pipeline đầy đủ
1. Vagrant tạo VM  
2. `common` → OS + runtime  
3. `master_init` → cluster  
4. `worker_join` → node Ready  
5. `deploy_core_build` → build image  
6. `load_custom_images` → load vào node  
7. `deploy_core` → deploy NF  
8. `label_nodes` → điều khiển placement  
9. `expose_gateway` → mở truy cập  

---

## 4 điều kiện để hệ thống chạy được
1. Image tồn tại trên đúng node  
2. `nodeSelector` đúng (centre/edge)  
3. Config (gateway/registrar) thống nhất  
4. Pod không crash  

---

## Debug nhanh (luôn dùng)
```
kubectl get pods -n etri6g -o wide
kubectl describe pod <pod>
kubectl logs <pod>
```

---

(Tài liệu rút gọn + giải thích module, giữ đầy đủ logic vận hành)
