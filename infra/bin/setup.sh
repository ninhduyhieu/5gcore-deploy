#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_DIR="$(dirname "$SCRIPT_DIR")"

echo "========================================="
echo "  ETRI B5GC Infrastructure Setup"
echo "========================================="
echo ""

echo "[1/3] Starting Vagrant cluster..."
cd "$INFRA_DIR"
vagrant up

echo ""
echo "[2/3] Verifying k3s cluster..."
vagrant ssh k3s-server -c "sudo k3s kubectl get nodes"

echo ""
echo "[3/3] Checking ETRI B5GC deployment status..."
vagrant ssh k3s-server -c "sudo k3s kubectl get pods -n etri6g" 2>/dev/null || echo "NFs not yet deployed. Argo CD is syncing..."

echo ""
echo "========================================="
echo "  Setup complete!"
echo "========================================="
echo ""
echo "  Argo CD UI:  http://192.168.56.10:30080"
echo "  Gateway:     http://192.168.56.10:30007"
echo "  Controller:  http://192.168.56.10:30008"
echo ""
echo "  Get Argo CD password:"
echo "    vagrant ssh k3s-server -c \"sudo k3s kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d\""
echo ""
