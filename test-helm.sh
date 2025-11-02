#!/bin/bash
rm -f *.tgz||true
curl -v http://127.0.0.1:5000/helm/jetstack/index.yaml
helm repo rm jetstack||true
helm repo add jetstack http://127.0.0.1:5000/helm/jetstack
helm fetch jetstack/cert-manager
ls -l *.tgz

rm -f *.tgz||true
curl -v http://127.0.0.1:5000/helm/cloudnative-pg/index.yaml

curl -v "http://127.0.0.1:5000/helm/cloudnative-pg/external/https/github.com/cloudnative-pg/charts/releases/download/plugin-barman-cloud-v0.1.0/plugin-barman-cloud-0.1.0.tgz"

helm repo rm cloudnativepg||true
helm repo rm cloudnative-pg||true
helm repo add cloudnative-pg http://127.0.0.1:5000/helm/cloudnative-pg
helm fetch cloudnative-pg/cnpg-operator
ls -l *.tgz
