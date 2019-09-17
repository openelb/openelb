FROM alpine
WORKDIR /
RUN apk add iptables
COPY agent .
ENTRYPOINT ["/agent"]