#!/usr/bin/env bash
set -euo pipefail

eval $(minikube docker-env)

docker build -t myapp-backend:latest -f build/Dockerfile .
docker build -t myapp-frontend:latest -f frontend/Dockerfile ./frontend

kubectl apply -k k8s/

kubectl wait --for=condition=ready deployment/frontend -n app --timeout=120s
kubectl wait --for=condition=ready deployment/app -n app --timeout=120s
kubectl wait --for=condition=complete job/migrate -n app --timeout=300s
