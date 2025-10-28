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

# Testing consistency during intterruption of pull
# podman rmi localhost:5000/dockerhub/postgres:latest||true
#
# 1. Launch server
# 2. Start pull: podman image pull  localhost:5000/dockerhub/postgres:latest  --tls-verify=false
# 3. During pull, kill server or client prematurely using CTRL+C
# 4. Restart server
# 5. Re-run pull command, it should return an error about mismatched digest and not start over
#
# Error: writing blob: storing blob to file "/var/tmp/storage948283329/3":
# happened during read: Digest did not match, expected
# sha256:23ed4c7c49ef8bac9af6c023d630098a6e6e3e601363ebe7620506124f62df8c,
# got sha256:33679430848de32c3d5c2ed1d5bd9b71fd3c2113dec30d3d383ecf70c65cdc94


