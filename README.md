# gobinrepo

`gobinrepo` is a lightweight Go‑based registry proxy that demonstrates, from its very first commit, the ability to **cache and serve container images** locally.

The design goal is to make repeated pulls dramatically faster and reduce redundant network traffic, while keeping the proxy simple and idiomatic.

---

## ✨ First Commit Feature

The initial proof of concept provided:

- Acts as a proxy for Docker/OCI images.
- Caches blobs under `/tmp/blobs/`.
- Serves cached content on repeated pulls.
- Demonstrates integration with `podman` as a client.

---

## 🔑 Built‑in Repository Keys

By default, `gobinrepo` starts with two built‑in repository configurations:

- **dockerhub** → `https://registry-1.docker.io`
- **quayio** → `https://quay.io`

This means you can immediately pull images through the proxy using either repoKey without any extra configuration.

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
