#!/bin/sh
# Copyright 2020 The KubeSphere Authors. All rights reserved.
# Use of this source code is governed by an Apache license
# that can be found in the LICENSE file.
# NOTICE: This script is a fork from klipper-lb v0.2.0 which uses Apache License 2.0

set -e

# FIXME: the NAT routing to LoadBalancer needs the /proc/sys/net/ipv4/ip_forward to be 1
# But in many cases this parameter is always 0
# Until now, there's no way to modify sys parameters in docker image, these sys parameters will always
# be set to default or as same as host at runtime, and can't be modified without priviledged run.
echo 1 > /proc/sys/net/ipv4/ip_forward