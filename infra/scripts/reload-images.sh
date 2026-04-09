#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_DIR="$(dirname "$SCRIPT_DIR")"

NF_BINARY_DIR="$INFRA_DIR/helm/etrib5gc-main/bin"
DOCKERFILE="$INFRA_DIR/docker/nf.Dockerfile"
CONTEXT="$INFRA_DIR"

NF="${1:-all}"

build_nf() {
	local nf="$1"
	if [ ! -f "$NF_BINARY_DIR/$nf" ]; then
		echo "SKIP: binary $nf not found in $NF_BINARY_DIR"
		return
	fi
	echo "Building b5gc-${nf}:latest ..."
	docker build \
		--build-arg B5GC_MODULE="$nf" \
		-t "b5gc-${nf}:latest" \
		-f "$DOCKERFILE" \
		"$CONTEXT"
}

export_images() {
	local nf="$1"
	echo "Exporting b5gc-${nf}:latest ..."
	docker save "b5gc-${nf}:latest" -o "$INFRA_DIR/artifacts/b5gc-${nf}.tar"
}

import_into_k3s() {
	local node="$1"
	local nf="$2"
	echo "Importing b5gc-${nf}:latest on $node ..."
	vagrant ssh "$node" -c "sudo k3s ctr images import /vagrant/artifacts/b5gc-${nf}.tar"
}

if [ "$NF" = "all" ]; then
	mkdir -p "$INFRA_DIR/artifacts"
	for bin in "$NF_BINARY_DIR"/*; do
		nf_name="$(basename "$bin")"
		build_nf "$nf_name"
		export_images "$nf_name"
	done
	echo ""
	echo "Importing images into k3s nodes..."
	for bin in "$NF_BINARY_DIR"/*; do
		nf_name="$(basename "$bin")"
		import_into_k3s "k3s-server" "$nf_name"
		import_into_k3s "k3s-agent1" "$nf_name"
		import_into_k3s "k3s-agent2" "$nf_name"
	done
else
	mkdir -p "$INFRA_DIR/artifacts"
	build_nf "$NF"
	export_images "$NF"
	for node in k3s-server k3s-agent1 k3s-agent2; do
		import_into_k3s "$node" "$NF"
	done
fi

echo ""
echo "Done. Restarting Argo CD sync..."
vagrant ssh k3s-server -c "sudo k3s kubectl apply -f /vagrant/argocd/templates/etrib5gc-app.yaml" 2>/dev/null || true
