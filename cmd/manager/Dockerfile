FROM alpine
WORKDIR /
RUN apk add iptables
ARG TARGETOS
ARG TARGETARCH
COPY bin/manager-${TARGETOS}-${TARGETARCH} /usr/local/bin/openelb-manager
COPY bin/gobgp-${TARGETOS}-${TARGETARCH} /usr/local/bin/gobgp
ENTRYPOINT ["openelb-manager"]
