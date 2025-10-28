# gobinrepo

`gobinrepo` is a lightweight Go‑based registry proxy that demonstrates, from its very first commit, the ability to **cache and serve container images** locally.

The initial feature is simple but powerful: once an image is pulled through `gobinrepo`, subsequent pulls are served from the local cache instead of re‑fetching from the upstream registry. This makes repeated pulls dramatically faster and avoids redundant network traffic.

---

## ✨ First Commit Feature

The first commit provides a working proof of concept:

- Acts as a proxy for Docker/OCI images.
- Caches blobs under `/tmp/blobs/`.
- Serves cached content on repeated pulls.
- Demonstrates integration with `podman` as a client.

---

## 🚀 Quick Demo

The following script shows the feature in action:

```bash
#!/bin/bash

# Remove any previously cached image
podman rmi localhost:5000/docker-remote/postgres:latest || true

# Clear cached blobs
rm -rf /tmp/blobs/

# First pull: fetched from upstream and cached
time podman image pull localhost:5000/docker-remote/postgres:latest --tls-verify=false

# Second pull: served from local cache (much faster)
time podman image pull localhost:5000/docker-remote/postgres:latest --tls-verify=false
