# Copyright 2020 The KubeSphere Authors. All rights reserved.
# Use of this source code is governed by an Apache license
# that can be found in the LICENSE file.

FROM golang:1.15 as porter-builder

COPY / /go/src/github.com/kubesphere/porter

WORKDIR /go/src/github.com/kubesphere/porter
RUN GO111MODULE=on CGO_ENABLED=0 go install -i -ldflags '-w -s' github.com/kubesphere/porter/cmd/...
RUN GO111MODULE=on CGO_ENABLED=0 go install -i -ldflags '-w -s' github.com/osrg/gobgp/cmd/gobgp

FROM alpine:3.9
RUN apk add --update ca-certificates iptables && update-ca-certificates
COPY --from=porter-builder /go/bin/agent /usr/local/bin/
COPY --from=porter-builder /go/bin/manager /usr/local/bin/
COPY --from=porter-builder /go/bin/gobgp /usr/local/bin/

CMD ["sh"]
