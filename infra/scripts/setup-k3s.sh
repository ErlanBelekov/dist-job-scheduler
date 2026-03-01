#!/usr/bin/env bash
set -euo pipefail

# Usage: ./setup-k3s.sh <server-ip>
# Installs k3s on a remote server and copies the kubeconfig locally.

SERVER_IP="${1:?Usage: ./setup-k3s.sh <server-ip>}"
SSH_KEY="${2:-$HOME/.ssh/id_rsa_personal}"
SSH="ssh -i ${SSH_KEY} -o StrictHostKeyChecking=accept-new root@${SERVER_IP}"

echo "==> Installing k3s on ${SERVER_IP}..."
$SSH "curl -sfL https://get.k3s.io | sh -"

echo "==> Waiting for k3s to be ready..."
$SSH "k3s kubectl wait --for=condition=Ready node --all --timeout=120s"

echo "==> Fetching kubeconfig..."
mkdir -p ~/.kube
$SSH "cat /etc/rancher/k3s/k3s.yaml" \
  | sed "s/127.0.0.1/${SERVER_IP}/g" \
  > ~/.kube/dist-scheduler.yaml

echo "==> Done! Set your kubeconfig:"
echo "  export KUBECONFIG=~/.kube/dist-scheduler.yaml"
echo "  kubectl get nodes"
