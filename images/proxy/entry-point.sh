#!/bin/sh
# Copyright 2021 The KubeSphere Authors. All rights reserved.
# Use of this source code is governed by an Apache license
# that can be found in the LICENSE file.
# NOTICE: This script is a fork from klipper-lb v0.2.0 which uses Apache License 2.0

# PROXY_ARGS is 4-tuple parameters split by space: <SVC_IP POD_PORT SVC_PORT SVC_PROTO>
# eg. PROXY_ARGS="192.0.0.1 80 80 tcp 192.0.0.2 3000 3000 udp" generates:
#     iptables -t nat -I PREROUTING ! -s 192.0.0.1/32 -p tcp --dport 80 -j DNAT --to 192.0.0.1:80
#     iptables -t nat -I POSTROUTING -d 192.0.0.1/32 -p tcp -j MASQUERADE
#     iptables -t nat -I PREROUTING ! -s 192.0.0.2/32 -p udp --dport 3000 -j DNAT --to 192.0.0.2:3000
#     iptables -t nat -I POSTROUTING -d 192.0.0.2/32 -p dup -j MASQUERADE
# This routes by NAT:
#     <host_ip:80>    ---->>  <svc_cluster_ip:80>    (tcp)
#     <host_ip:3000>  ---->>  <svc_cluster_ip:3000>  (udp)

set -e

echo $PROXY_ARGS | xargs -n4 -t sh -c 'iptables -t nat -I PREROUTING ! -s $1/32 -p $4 --dport $2 -j DNAT --to $1:$3' sh
echo $PROXY_ARGS | xargs -n4 -t sh -c 'iptables -t nat -I POSTROUTING -d $1/32 -p $4 -j MASQUERADE' sh

while true
do
        sleep 2048d
done
