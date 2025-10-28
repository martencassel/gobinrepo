#!/bin/bash

podman rmi localhost:5000/dockerhub/postgres:latest||true
rm -rf /tmp/blobs/
time podman image pull  localhost:5000/dockerhub/postgres:latest  --tls-verify=false
time podman image pull  localhost:5000/dockerhub/postgres:latest  --tls-verify=false

podman rmi localhost:5000/quayio/argoproj/argocd:latest||true
time podman image pull  localhost:5000/quayio/argoproj/argocd:latest  --tls-verify=false
time podman image pull  localhost:5000/quayio/argoproj/argocd:latest  --tls-verify=false

podman rmi localhost:5000/ghcr/github/super-linter:slim-v5.0.0||true
time podman image pull localhost:5000/ghcr/github/super-linter:slim-v5.0.0 --tls-verify=false
time podman image pull localhost:5000/ghcr/github/super-linter:slim-v5.0.0 --tls-verify=false

podman rmi localhost:5000/gcr/distroless/static:latest||true
time podman image pull localhost:5000/gcr/distroless/static:latest --tls-verify=false
time podman image pull localhost:5000/gcr/distroless/static:latest --tls-verify=false

podman rmi localhost:5000/mcr.microsoft.com/powershell:latest||true
time podman image pull localhost:5000/mcr/powershell:latest --tls-verify=false
time podman image pull localhost:5000/mcr/powershell:latest --tls-verify=false

podman rmi localhost:5000/publicecr/nginx/nginx:latest||true
time podman image pull localhost:5000/publicecr/nginx/nginx:latest --tls-verify=false
time podman image pull localhost:5000/publicecr/nginx/nginx:latest --tls-verify=false

podman rmi localhost:5000/icr/appcafe/open-liberty:latest||true
time podman pull localhost:5000/icr/appcafe/open-liberty:latest --tls-verify=false
time podman pull localhost:5000/icr/appcafe/open-liberty:latest --tls-verify=false

podman rmi localhost:5000/ocir/os/oraclelinux:8-slim||true
time podman pull localhost:5000/ocir/os/oraclelinux:8-slim --tls-verify=false
time podman pull localhost:5000/ocir/os/oraclelinux:8-slim --tls-verify=false

podman rmi localhost:5000/nvcr/nvidia/cuda:12.2.0-base-ubuntu22.04||true
time podman pull localhost:5000/nvcr/nvidia/cuda:12.2.0-base-ubuntu22.04 --tls-verify=false
time podman pull localhost:5000/nvcr/nvidia/cuda:12.2.0-base-ubuntu22.04 --tls-verify=false

podman rmi localhost:5000/gitlab/gitlab-org/gitlab-runner:alpine||true
time podman pull localhost:5000/gitlab/gitlab-org/gitlab-runner:alpine --tls-verify=false
time podman pull localhost:5000/gitlab/gitlab-org/gitlab-runner:alpine --tls-verify=false

podman rmi localhost:5000/redhat/ubi8/ubi-minimal:latest||true
time podman pull localhost:5000/redhat/ubi8/ubi-minimal:latest --tls-verify=false
time podman pull localhost:5000/redhat/ubi8/ubi-minimal:latest --tls-verify=false
