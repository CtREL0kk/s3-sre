#!/usr/bin/env bash
set -euo pipefail

minikube image build -t myapp-backend:latest -f build/Dockerfile .
minikube image build -t myapp-frontend:latest -f Dockerfile frontend/
minikube image build -t myapp-prober:latest -f Dockerfile prober/

kubectl apply -k k8s/
kubectl wait --for=condition=complete job/migrate -n app --timeout=60s

kubectl rollout status deployment/app -n app --timeout=60s
kubectl rollout status deployment/frontend -n app --timeout=60s

kubectl apply -k k8s/observability/
make metrics
kubectl rollout status deployment/victoria-logs -n monitoring --timeout=120s
kubectl rollout status daemonset/vector -n monitoring --timeout=120s
kubectl port-forward -n ingress-nginx service/ingress-nginx-controller 8080:80
