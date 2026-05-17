#!/bin/bash
set -e

# Đọc từ env vars (truyền từ CI/CD), fallback về $HOME nếu chạy standalone
INFRA_DIR="${INFRA_DIR:-$HOME/5gcore-deploy}"
STORMSIM_DIR="${STORMSIM_DIR:-$HOME/Downloads/stormsim}"
GTP5G_DIR="${GTP5G_DIR:-$HOME/gtp5g}"
UPF_DIR="${UPF_DIR:-$HOME/Downloads/etrib5gc}"

KUBECONFIG_PATH="$INFRA_DIR/kubeconfig"
STORMSIM_CONFIG="$STORMSIM_DIR/config/2slice-test.yml"
UPF_CONFIG="$UPF_DIR/config/upf.json"
NAMESPACE="etri6g"

export KUBECONFIG="$KUBECONFIG_PATH"

echo ""
echo "================================================"
echo "  5G Core Test Runner"
echo "================================================"

# ── Bước 1: Kiểm tra & load gtp5g kernel module ──────
echo ""
echo "[1/6] Kiểm tra gtp5g kernel module..."
KERNEL_VER=$(uname -r)
echo "      Kernel: $KERNEL_VER"

if ! lsmod | grep -q gtp5g; then
  echo "      gtp5g chưa load, thử modprobe..."
  if ! sudo modprobe gtp5g 2>/dev/null; then
    echo "      modprobe fail, build lại cho kernel $KERNEL_VER..."
    cd "$GTP5G_DIR"
    make clean
    make
    sudo insmod gtp5g.ko
  fi
fi

if lsmod | grep -q gtp5g; then
  echo "      ✓ gtp5g loaded"
else
  echo "      ✗ FATAL: gtp5g không load được, dừng test"
  exit 1
fi

# ── Bước 2: Kiểm tra & khởi động UPF ─────────────────
echo ""
echo "[2/6] Kiểm tra UPF..."
if pgrep -x "upf" > /dev/null; then
  echo "      UPF đang chạy (PID: $(pgrep -x upf))"

  # Kiểm tra UPF đã register vào Gateway chưa
  GW_LOG=$(kubectl logs deployment/gateway-dep -n "$NAMESPACE" --tail=50 2>/dev/null || true)
  if echo "$GW_LOG" | grep -qi "upf"; then
    echo "      ✓ UPF đã register vào Gateway"
  else
    echo "      ⚠ UPF đang chạy nhưng chưa thấy trong Gateway log, đợi thêm..."
    sleep 10
  fi
else
  echo "      UPF chưa chạy, khởi động..."
  cd "$UPF_DIR"
  sudo nohup bin/upf -c "$UPF_CONFIG" > /tmp/upf.log 2>&1 &
  echo "      Đợi UPF register vào Gateway (30s)..."
  sleep 30

  if pgrep -x "upf" > /dev/null; then
    echo "      ✓ UPF started (PID: $(pgrep -x upf))"
  else
    echo "      ✗ FATAL: UPF không start được"
    echo "      Log: $(tail -5 /tmp/upf.log)"
    exit 1
  fi
fi

# ── Bước 3: Chờ các pod K8s sẵn sàng ─────────────────
echo ""
echo "[3/6] Chờ các pod etri6g sẵn sàng..."
kubectl wait --for=condition=ready pod \
  -l app=pran -n "$NAMESPACE" --timeout=120s
kubectl wait --for=condition=ready pod \
  -l app=amf -n "$NAMESPACE" --timeout=120s
kubectl wait --for=condition=ready pod \
  -l app=smf -n "$NAMESPACE" --timeout=120s
echo "      ✓ Pods sẵn sàng"

# ── Bước 4: Seed toàn bộ subscriber vào MongoDB ───────
echo ""
echo "[4/6] Seed subscriber data vào MongoDB..."
MONGO_POD=$(kubectl get pod -n "$NAMESPACE" -l app=udr-mongo \
  -o jsonpath='{.items[0].metadata.name}')
echo "      Mongo pod: $MONGO_POD"

kubectl exec -n "$NAMESPACE" "$MONGO_POD" -- mongosh --quiet --eval "
var db_etri = db.getSiblingDB('etrib5gc');
var KEY = '99b9d52c3f5fc0fb2bb40700600ed36c';
var OPC = '74e99a5681b03235688b0fe280162f4d';
var PLMN = '001-01';

db_etri.authsub.deleteMany({});
db_etri.amsub.deleteMany({});
db_etri.smsub.deleteMany({});
db_etri.smfsel.deleteMany({});

for (var i = 1; i <= 10; i++) {
  var msin = ('0000000000' + i).slice(-10);
  var imsi = 'imsi-00101' + msin;
  var plmnKey = imsi + '_' + PLMN;
  var sd = (i <= 5) ? '010203' : '543210';

  db_etri.authsub.insertOne({
    _id: imsi, ueId: imsi, plmnId: PLMN,
    dat: {
      sequencenumber: {sqn: '000000000000', sqnscheme: 'NON_TIME_BASED', lastindexes: {}},
      authenticationmethod: '5G_AKA',
      encpermanentkey: KEY,
      encopckey: OPC,
      authenticationmanagementfield: '8000'
    }
  });

  db_etri.amsub.insertOne({
    _id: plmnKey, ueId: imsi, plmnId: PLMN,
    dat: {
      nssai: {
        defaultsinglenssais: [{sd: sd, sst: 1}],
        singlenssais: [{sd: '010203', sst: 1}, {sd: '543210', sst: 1}]
      }
    }
  });

  db_etri.smsub.insertOne({
    _id: plmnKey, ueId: imsi, plmnId: PLMN,
    dat: {
      sessionmanagementsubscriptiondata: [{
        singleNssai: {sst: 1, sd: sd},
        dnnConfigurations: {
          internet: {
            pduSessionTypes: {defaultSessionType: 'IPV4', allowedSessionTypes: ['IPV4']},
            sscModes: {defaultSscMode: 'SSC_MODE_1', allowedSscModes: ['SSC_MODE_2', 'SSC_MODE_3']},
            '5gQosProfile': {
              '5qi': 9,
              arp: {preemptCap: 'NOT_PREEMPT', preemptVuln: 'NOT_PREEMPTABLE', priorityLevel: 8},
              priorityLevel: 8
            },
            sessionAmbr: {downlink: '1000 Mbps', uplink: '1000 Mbps'}
          }
        }
      }]
    }
  });

  db_etri.smfsel.insertOne({
    _id: plmnKey, ueId: imsi, plmnId: PLMN,
    dat: {
      subscribedSnssaiInfos: {
        '01010203': {dnnInfos: [{dnn: 'internet'}]},
        '01543210': {dnnInfos: [{dnn: 'internet'}]}
      }
    }
  });
}

print('authsub: ' + db_etri.authsub.countDocuments());
print('amsub:   ' + db_etri.amsub.countDocuments());
print('smsub:   ' + db_etri.smsub.countDocuments());
print('smfsel:  ' + db_etri.smfsel.countDocuments());
"
echo "      ✓ Subscriber data seeded (10 UE)"

# ── Bước 5: Restart AMF/DAMF rồi lấy PRAN IP ─────────
echo ""
echo "[5/6] Restart AMF & DAMF để clear cache..."
kubectl rollout restart deployment/amf-10-100-agent1-dep \
  deployment/amf-10-100-agent2-dep deployment/damf-dep \
  -n "$NAMESPACE"
kubectl rollout status deployment/amf-10-100-agent1-dep \
  -n "$NAMESPACE" --timeout=60s
kubectl rollout status deployment/amf-10-100-agent2-dep \
  -n "$NAMESPACE" --timeout=60s
kubectl rollout status deployment/damf-dep \
  -n "$NAMESPACE" --timeout=60s
echo "      ✓ AMF & DAMF ready"

# Lấy PRAN IP SAU khi restart xong (pod có thể đổi IP nếu restart)
PRAN_IP=$(kubectl get pod -n "$NAMESPACE" -l app=pran \
  -o jsonpath='{.items[0].status.podIP}')
echo "      PRAN IP: $PRAN_IP"
sed -i "s/- ip: \".*\"/- ip: \"$PRAN_IP\"/" "$STORMSIM_CONFIG"
echo "      ✓ StormSIM config updated: amfif.ip = $PRAN_IP"

# ── Bước 6: Chạy StormSIM ─────────────────────────────
echo ""
echo "[6/6] Chạy StormSIM test..."
echo "      Config: $STORMSIM_CONFIG"
echo ""
cd "$STORMSIM_DIR"
sudo ./bin/emulator -c config/2slice-test.yml 2>&1 | \
  grep -E "PDUSession_Active|5GMM_Registered|RegistrationReject|ErrorIndication|WARN|ERROR|INFO.*UE|INFO.*Starting|INFO.*Simulation"

echo ""
echo "================================================"
echo "  Test hoàn tất"
echo "================================================"
