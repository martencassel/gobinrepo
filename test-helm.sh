#!/bin/bash
rm -f *.tgz||true
curl -v http://127.0.0.1:5000/helm/jetstack/index.yaml
helm repo rm jetstack||true
helm repo add jetstack http://127.0.0.1:5000/helm/jetstack
helm fetch jetstack/cert-manager
ls -l *.tgz
