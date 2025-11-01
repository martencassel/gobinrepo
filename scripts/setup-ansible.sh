#!/bin/bash

# 1. Choose Upstream Repos
#
# Community: https://galaxy.ansible.com/api/, Collection: community.general
# Certified: https://console.redhat.com/api/automation-hub/, Collection: redhat.rhel_system_roles

# 2. Configure Your Proxy Cache
#
# Set up your repository_remote with:
# url: pointing to the upstream
# proxy_mode: enabled
# policy: on_demand or immediate depending on your caching strategy

# 3. Client Configuration
#
# In your ansible.cfg:
#
# [galaxy]
# server_list = my_proxy
#
# [galaxy_server.my_proxy]
# url=https://<your-proxy-host>/api/

# 4. Test commands
#
# ansible-galaxy collection install community.general
#
# ansible-galaxy collection install community.general
#
# ansible-galaxy collection install community.general:3.8.0
#
# ansible-galaxy collection install redhat.rhel_system_roles



