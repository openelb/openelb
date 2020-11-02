FROM alpine
WORKDIR /
RUN apk add iptables
ARG TARGETOS
ARG TARGETARCH
COPY bin/manager-${TARGETOS}-${TARGETARCH} manager
ENTRYPOINT ["/manager"]
