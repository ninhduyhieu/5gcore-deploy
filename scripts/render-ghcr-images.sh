#!/usr/bin/env bash
set -euo pipefail

OWNER="$1"
TAG="$2"
OUT_DIR="${3:-rendered}"

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"
cp k8s/*.yaml "$OUT_DIR"/

# Replace local images with GHCR images
sed -i "s|image: b5gc-controller:latest|image: ghcr.io/${OWNER}/b5gc-controller:${TAG}|g" "$OUT_DIR/ctrl.yaml"
sed -i "s|image: b5gc-gateway:latest|image: ghcr.io/${OWNER}/b5gc-gateway:${TAG}|g" "$OUT_DIR/gw.yaml"
sed -i "s|image: b5gc-nsm:latest|image: ghcr.io/${OWNER}/b5gc-nsm:${TAG}|g" "$OUT_DIR/nsm.yaml"

sed -i "s|image: b5gc-ausf:latest|image: ghcr.io/${OWNER}/b5gc-ausf:${TAG}|g" "$OUT_DIR/deploy-other.yaml"
sed -i "s|image: b5gc-udm:latest|image: ghcr.io/${OWNER}/b5gc-udm:${TAG}|g" "$OUT_DIR/deploy-other.yaml"
sed -i "s|image: b5gc-pcf:latest|image: ghcr.io/${OWNER}/b5gc-pcf:${TAG}|g" "$OUT_DIR/deploy-other.yaml"
sed -i "s|image: b5gc-damf:latest|image: ghcr.io/${OWNER}/b5gc-damf:${TAG}|g" "$OUT_DIR/deploy-other.yaml"
sed -i "s|image: b5gc-amf:latest|image: ghcr.io/${OWNER}/b5gc-amf:${TAG}|g" "$OUT_DIR/deploy-other.yaml"
sed -i "s|image: b5gc-smf:latest|image: ghcr.io/${OWNER}/b5gc-smf:${TAG}|g" "$OUT_DIR/deploy-other.yaml"

sed -i "s|image: b5gc-pran:latest|image: ghcr.io/${OWNER}/b5gc-pran:${TAG}|g" "$OUT_DIR/pran.yaml"

# Đổi imagePullPolicy sang Always để đảm bảo pull image mới từ GHCR
for f in "$OUT_DIR"/*.yaml; do
  sed -i "s|imagePullPolicy: IfNotPresent|imagePullPolicy: Always|g" "$f"
done

echo "Rendered manifests in: $OUT_DIR"
grep -R "image:" "$OUT_DIR" || true