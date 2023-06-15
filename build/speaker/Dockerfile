# Build
FROM golang:1.19 as build_context

ENV OUTDIR=/out
RUN mkdir -p ${OUTDIR}/usr/local/bin/

WORKDIR /workspace
ADD . /workspace/

RUN make speaker
RUN go build -o bin/cmd/gobgp github.com/osrg/gobgp/cmd/gobgp
RUN mv /workspace/bin/cmd/speaker ${OUTDIR}/usr/local/bin/openelb-speaker
RUN mv /workspace/bin/cmd/gobgp ${OUTDIR}/usr/local/bin/gobgp


##############
# Final image
#############
FROM alpine:3.17

COPY --from=build_context /out/ /
WORKDIR /

ENTRYPOINT ["openelb-speaker"]

