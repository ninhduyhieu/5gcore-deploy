#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
INFRA_DIR="$(dirname "$SCRIPT_DIR")"
DOCKERFILE="${INFRA_DIR}/docker/nf.Dockerfile"
BIN_DIR="${INFRA_DIR}/helm/etrib5gc-main/bin"

ALL_NFS=(amf ausf controller damf gateway nsm pcf pran sepp smf udm upf)

build_nf() {
	local nf="$1"
	if [ ! -f "${BIN_DIR}/${nf}" ]; then
		echo "SKIP: binary not found for ${nf}"
		return
	fi
	echo "BUILD: b5gc-${nf}:latest"
	docker build --build-arg B5GC_MODULE="${nf}" -t "b5gc-${nf}:latest" -f "${DOCKERFILE}" "${INFRA_DIR}"
}

if [ $# -eq 0 ] || [ "$1" = "all" ]; then
	for nf in "${ALL_NFS[@]}"; do
		build_nf "${nf}"
	done
else
	build_nf "$1"
fi

echo "DONE"
