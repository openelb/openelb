# Copy the controller-manager into a thin image
FROM alpine
RUN apk add iptables
WORKDIR /
COPY manager .
ENTRYPOINT ["/manager"]
