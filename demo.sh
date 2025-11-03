#!/bin/bash

# Docker Remote

podman rmi localhost:5000/dockerhub/postgres:latest||true

time podman image pull localhost:5000/dockerhub/postgres:latest --tls-verify=false
time podman image pull localhost:5000/dockerhub/postgres:latest --tls-verify=false

# Helm OCI

helm pull oci://localhost:5000/quayio/strimzi-helm/strimzi-kafka-operator


# Helm Legacy

helm repo add jetstack http://127.0.0.1:5000/helm/jetstack
helm fetch jetstack/cert-manager

helm repo add cloudnative-pg http://127.0.0.1:5000/helm/cloudnative-pg
helm fetch cloudnative-pg/cnpg-operator

# Debian Remote

./scripts/get-ip.sh
./run-ip.sh
cd ./scripts/ && make

