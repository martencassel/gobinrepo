# gobinrepo

`gobinrepo` is a lightweight Go‑based artifact caching proxy that demonstrates, the ability to **cache and serve container images, helm charts, and debian packages** locally.

The design goal is to make repeated downloads dramatically faster and reduce redundant network traffic.

---

- Acts as a proxy for Docker/OCI images, Helm charts (classic), Debian repositories.
- Caches blobs to the local filesystem.
- Serves cached content on repeated pulls.
- Demonstrates integration with `podman` as a client.
- Authenticate to private registries such as docker.io subscription
- If an image pull is interrupted, incomplete downloads are never visible under the final path.

---

# Configuration and Overrides

Config file (config.yaml) You can define additional repositories or override the built‑ins by editing your config file.
The file is loaded at startup and provides the authoritative list of repositories.

---

## 🚀 Quick Demo

The following script shows the feature in action with both built‑in repoKeys:

```bash
#!/bin/bash

# --- Docker Hub example ---

# Remove any previously cached image
podman rmi localhost:5000/dockerhub/library/postgres:latest || true

# Clear cached blobs
rm -rf /tmp/blobs/

# First pull: fetched from Docker Hub and cached
time podman image pull localhost:5000/dockerhub/library/postgres:latest --tls-verify=false

# Second pull: served from local cache (much faster)
time podman image pull localhost:5000/dockerhub/library/postgres:latest --tls-verify=false


# --- Quay.io example ---

# Remove any previously cached image
podman rmi localhost:5000/quayio/argoproj/argocd:latest || true

# Clear cached blobs
rm -rf /tmp/blobs/

# First pull: fetched from Quay.io and cached
time podman image pull localhost:5000/quayio/argoproj/argocd:latest --tls-verify=false

# Second pull: served from local cache (much faster)
time podman image pull localhost:5000/quayio/argoproj/argocd:latest --tls-verify=false
```

---

## 🌍 Additional Supported Registries

Beyond Docker Hub and Quay.io, `gobinrepo` now supports pulling and caching from a wide range of registries.
The following examples demonstrate repeated pulls (first from remote, then from cache):

```bash
# GitHub Container Registry
podman rmi localhost:5000/ghcr/github/super-linter:slim-v5.0.0 || true
time podman image pull localhost:5000/ghcr/github/super-linter:slim-v5.0.0 --tls-verify=false
time podman image pull localhost:5000/ghcr/github/super-linter:slim-v5.0.0 --tls-verify=false

# Google Container Registry (Distroless)
podman rmi localhost:5000/gcr/distroless/static:latest || true
time podman image pull localhost:5000/gcr/distroless/static:latest --tls-verify=false
time podman image pull localhost:5000/gcr/distroless/static:latest --tls-verify=false

# Microsoft Container Registry
podman rmi localhost:5000/mcr.microsoft.com/powershell:latest || true
time podman image pull localhost:5000/mcr/powershell:latest --tls-verify=false
time podman image pull localhost:5000/mcr/powershell:latest --tls-verify=false

# AWS Public ECR
podman rmi localhost:5000/publicecr/nginx/nginx:latest || true
time podman image pull localhost:5000/publicecr/nginx/nginx:latest --tls-verify=false
time podman image pull localhost:5000/publicecr/nginx/nginx:latest --tls-verify=false

# IBM Cloud Container Registry
podman rmi localhost:5000/icr/appcafe/open-liberty:latest || true
time podman pull localhost:5000/icr/appcafe/open-liberty:latest --tls-verify=false
time podman pull localhost:5000/icr/appcafe/open-liberty:latest --tls-verify=false

# Oracle Cloud Infrastructure Registry
podman rmi localhost:5000/ocir/os/oraclelinux:8-slim || true
time podman pull localhost:5000/ocir/os/oraclelinux:8-slim --tls-verify=false
time podman pull localhost:5000/ocir/os/oraclelinux:8-slim --tls-verify=false

# NVIDIA NGC
podman rmi localhost:5000/nvcr/nvidia/cuda:12.2.0-base-ubuntu22.04 || true
time podman pull localhost:5000/nvcr/nvidia/cuda:12.2.0-base-ubuntu22.04 --tls-verify=false
time podman pull localhost:5000/nvcr/nvidia/cuda:12.2.0-base-ubuntu22.04 --tls-verify=false

# GitLab Container Registry
podman rmi localhost:5000/gitlab/gitlab-org/gitlab-runner:alpine || true
time podman pull localhost:5000/gitlab/gitlab-org/gitlab-runner:alpine --tls-verify=false
time podman pull localhost:5000/gitlab/gitlab-org/gitlab-runner:alpine --tls-verify=false

# Red Hat Container Catalog
podman rmi localhost:5000/redhat/ubi8/ubi-minimal:latest || true
time podman pull localhost:5000/redhat/ubi8/ubi-minimal:latest --tls-verify=false
time podman pull localhost:5000/redhat/ubi8/ubi-minimal:latest --tls-verify=false
```
