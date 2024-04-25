# Build
FROM golang:1.21 as build_context
ENV OUTDIR=/out
RUN mkdir -p ${OUTDIR}/usr/local/bin/

WORKDIR /workspace
ADD . /workspace/

RUN make speaker
RUN mv /workspace/bin/cmd/speaker ${OUTDIR}/usr/local/bin/openelb-speaker
RUN GO111MODULE=on CGO_ENABLED=0 go install -ldflags '-w -s' github.com/osrg/gobgp/cmd/gobgp
RUN mv /go/bin/gobgp ${OUTDIR}/usr/local/bin/gobgp



##############
# Final image
#############
FROM alpine

RUN apk add --update --no-cache keepalived
COPY --from=build_context /out/ /
ADD build/speaker/keepalived.tmpl /
ADD build/speaker/keepalived-check.sh /
WORKDIR /

ENTRYPOINT ["openelb-speaker"]

