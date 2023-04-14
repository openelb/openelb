# Copyright 2020 The KubeSphere Authors. All rights reserved.
# Use of this source code is governed by an Apache license
# that can be found in the LICENSE file.

FROM golang:1.16 as openelb-builder

COPY / /go/src/github.com/openelb/openelb

WORKDIR /go/src/github.com/openelb/openelb
RUN GO111MODULE=on CGO_ENABLED=0 go install -i -ldflags '-w -s' github.com/openelb/openelb/cmd/...
RUN GO111MODULE=on CGO_ENABLED=0 go install -i -ldflags '-w -s' github.com/osrg/gobgp/cmd/gobgp

FROM alpine:3.9
RUN apk add --update ca-certificates iptables && update-ca-certificates
COPY --from=openelb-builder /go/bin/agent /usr/local/bin/openelb-agent
COPY --from=openelb-builder /go/bin/manager /usr/local/bin/openelb-manager
COPY --from=openelb-builder /go/bin/gobgp /usr/local/bin/gobgp

CMD ["sh"]
