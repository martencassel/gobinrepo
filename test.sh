#!/bin/bash

podman rmi localhost:5000/docker-remote/postgres:latest|true

rm -rf /tmp/blobs/

time podman image pull  localhost:5000/docker-remote/postgres:latest  --tls-verify=false

time podman image pull  localhost:5000/docker-remote/postgres:latest  --tls-verify=false
