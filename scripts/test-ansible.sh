#!/bin/bash

export http_proxy=http://127.0.0.1:9090
export https_proxy=http://127.0.0.1:9090

mitmproxy --listen-port 9090 -k 

ansible-galaxy role install geerlingguy.apache

