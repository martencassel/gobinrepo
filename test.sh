#!/bin/bash

podman rmi localhost:5000/dockerhub/postgres:latest|true
rm -rf /tmp/blobs/
time podman image pull  localhost:5000/dockerhub/postgres:latest  --tls-verify=false
time podman image pull  localhost:5000/dockerhub/postgres:latest  --tls-verify=false

podman rmi localhost:5000/quayio/argoproj/argocd:latest|true
time podman image pull  localhost:5000/quayio/argoproj/argocd:latest  --tls-verify=false
time podman image pull  localhost:5000/quayio/argoproj/argocd:latest  --tls-verify=false

