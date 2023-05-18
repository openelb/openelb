# Copyright 2020 The KubeSphere Authors. All rights reserved.
# Use of this source code is governed by an Apache license
# that can be found in the LICENSE file.

FROM golang:1.19 as openelb-builder

COPY / /go/src/github.com/openelb/openelb

WORKDIR /go/src/github.com/openelb/openelb
RUN GO111MODULE=on CGO_ENABLED=0 go install -i -ldflags '-w -s' github.com/openelb/openelb/cmd/...
RUN GO111MODULE=on CGO_ENABLED=0 go install -i -ldflags '-w -s' github.com/osrg/gobgp/cmd/gobgp

FROM alpine:3.17
RUN apk add --update ca-certificates iptables && update-ca-certificates
COPY --from=openelb-builder /go/bin/controller /usr/local/bin/openelb-controller
# COPY --from=openelb-builder /go/bin/speaker /usr/local/bin/openelb-speaker
# COPY --from=openelb-builder /go/bin/apiserver /usr/local/bin/openelb-apiserver
COPY --from=openelb-builder /go/bin/gobgp /usr/local/bin/gobgp

CMD ["sh"]
