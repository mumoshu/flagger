#!/usr/bin/env bash

set -o errexit

REPO_ROOT=$(git rev-parse --show-toplevel)
HELM_VERSION=v2.15.1
KIND_VERSION=v0.5.1

if [[ "$1" ]]; then
  KIND_VERSION=$1
fi

echo ">>> Installing kubectl"
curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl && \
chmod +x kubectl && \
sudo mv kubectl /usr/local/bin/

echo ">>> Installing kind"
curl -sSLo kind "https://github.com/kubernetes-sigs/kind/releases/download/$KIND_VERSION/kind-linux-amd64"
chmod +x kind
sudo mv kind /usr/local/bin/kind

echo ">>> Creating kind cluster"
kind create cluster --wait 5m #--image kindest/node:v1.16.2

export KUBECONFIG="$(kind get kubeconfig-path --name="kind")"
kubectl get pods --all-namespaces

echo ">>> Installing Helm"
curl -sSL https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz | tar xz && sudo mv linux-amd64/helm /usr/local/bin/ && rm -rf linux-amd64

echo '>>> Installing Tiller'
kubectl --namespace kube-system create sa tiller
kubectl create clusterrolebinding tiller-cluster-rule --clusterrole=cluster-admin --serviceaccount=kube-system:tiller
helm init --service-account tiller --upgrade --wait
