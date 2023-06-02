# Copyright 2020 The KubeSphere Authors. All rights reserved.
# Use of this source code is governed by an Apache license
# that can be found in the LICENSE file.

FROM alpine
WORKDIR /
COPY images/forward/entry-point.sh /entry-point.sh
RUN apk update && apk add iptables && \
    chmod +x /entry-point.sh
CMD [ "/entry-point.sh" ]
