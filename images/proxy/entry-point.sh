#!/bin/sh
# Copyright 2021 The KubeSphere Authors. All rights reserved.
# Use of this source code is governed by an Apache license
# that can be found in the LICENSE file.
# NOTICE: This script is a fork from klipper-lb v0.2.0 which uses Apache License 2.0

iptables -t nat -I PREROUTING ! -s ${SVC_IP}/32 -p tcp --dport ${POD_PORT} -j DNAT --to ${SVC_IP}:${SVC_PORT}
iptables -t nat -I POSTROUTING -d ${SVC_IP}/32 -p tcp -j MASQUERADE

while true
do
        sleep 2048d
done
