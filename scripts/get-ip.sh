#!/bin/bash

# Find primary interface (you can hardcode if needed)
PRIMARY_IF=$(ip route | grep default | awk '{print $5}')
echo "Primary interface: $PRIMARY_IF"

# Base IP range (adjust to match your subnet)
BASE="192.168.1"
START=100
END=200

# Find used IPs
USED_IPS=$(ip addr show "$PRIMARY_IF" | grep "inet " | awk '{print $2}' | cut -d/ -f1)

# Find a free IP
for i in $(seq $START $END); do
    IP="$BASE.$i"
    if ! echo "$USED_IPS" | grep -q "$IP"; then
        echo "Allocating IP: $IP"
        sudo ip addr add "$IP/24" dev "$PRIMARY_IF"
        echo "$IP" > .static_ip
        exit 0
    fi
done

echo "No free IPs found in range $BASE.$START-$END"
exit 1
