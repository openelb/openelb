FROM alpine

EXPOSE 179

RUN apk add curl && \
    curl -SL https://github.com/osrg/gobgp/releases/download/v2.23.0/gobgp_2.23.0_linux_amd64.tar.gz | tar xvz -C /usr/local/bin/

CMD gobgpd -l debug2