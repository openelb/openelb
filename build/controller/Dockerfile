# Build
FROM golang:1.21 as build_context

ENV OUTDIR=/out
RUN mkdir -p ${OUTDIR}/usr/local/bin/

WORKDIR /workspace
ADD . /workspace/

RUN make controller
RUN mv /workspace/bin/cmd/controller ${OUTDIR}/usr/local/bin/openelb-controller

##############
# Final image
#############
FROM alpine

COPY --from=build_context /out/ /
WORKDIR /

EXPOSE 443 50052
ENTRYPOINT ["openelb-controller"]

