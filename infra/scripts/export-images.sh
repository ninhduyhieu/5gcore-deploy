#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
INFRA_DIR="$(dirname "$SCRIPT_DIR")"
ARTIFACTS_DIR="${INFRA_DIR}/artifacts"

ALL_NFS=(amf ausf controller damf gateway nsm pcf pran sepp smf udm upf)

mkdir -p "${ARTIFACTS_DIR}"

export_nf() {
	local nf="$1"
	local image="b5gc-${nf}:latest"
	local tar="${ARTIFACTS_DIR}/b5gc-${nf}.tar"
	if docker image inspect "${image}" >/dev/null 2>&1; then
		echo "EXPORT: ${image} -> ${tar}"
		docker save -o "${tar}" "${image}"
	else
		echo "SKIP: ${image} not found"
	fi
}

if [ $# -eq 0 ] || [ "$1" = "all" ]; then
	for nf in "${ALL_NFS[@]}"; do
		export_nf "${nf}"
	done
else
	export_nf "$1"
fi

echo "DONE"
