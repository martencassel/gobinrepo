# gobinrepo

A fast caching proxy for container images, Helm charts, and Debian packages.

## What it does

- **Container images**: Pull from any registry, cache locally
- **Helm charts**: Both OCI and legacy HTTP repositories
- **Debian packages**: Mirror and cache apt repositories
- **Speed**: Second pulls are lightning fast from local cache

## Quick examples

```bash
# Container images - first pull downloads, second is instant
podman pull localhost:5000/dockerhub/postgres:latest --tls-verify=false
podman pull localhost:5000/dockerhub/postgres:latest --tls-verify=false

# Helm OCI charts
helm pull oci://localhost:5000/quayio/strimzi-helm/strimzi-kafka-operator

# Helm legacy repos
helm repo add jetstack http://127.0.0.1:5000/helm/jetstack
helm fetch jetstack/cert-manager

# Debian packages
echo "deb [trusted=yes] http://localhost:8080/debian/debian stable main" > /etc/apt/sources.list
apt update && apt install emacs
```

That's it. One proxy, three package types, much faster builds.
