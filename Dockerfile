# Copyright 2020 The KubeSphere Authors. All rights reserved.
# Use of this source code is governed by an Apache license
# that can be found in the LICENSE file.

FROM golang:1.15 as porter-builder

COPY / /go/src/github.com/kubesphere/porterlb

WORKDIR /go/src/github.com/kubesphere/porterlb
RUN GO111MODULE=on CGO_ENABLED=0 go install -i -ldflags '-w -s' github.com/kubesphere/porterlb/cmd/...
RUN GO111MODULE=on CGO_ENABLED=0 go install -i -ldflags '-w -s' github.com/osrg/gobgp/cmd/gobgp

FROM alpine:3.9
RUN apk add --update ca-certificates iptables && update-ca-certificates
COPY --from=porter-builder /go/bin/agent /usr/local/bin/porter-agent
COPY --from=porter-builder /go/bin/manager /usr/local/bin/porter-manager
COPY --from=porter-builder /go/bin/gobgp /usr/local/bin/gobgp

CMD ["sh"]
