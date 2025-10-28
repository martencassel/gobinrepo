#!/bin/bash

podman rmi localhost:5000/dockerhub/postgres:latest||true
time podman image pull  localhost:5000/dockerhub/postgres:latest  --tls-verify=false
time podman image pull  localhost:5000/dockerhub/postgres:latest  --tls-verify=false


