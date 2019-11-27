FROM ubuntu

LABEL author="magicsong@yunify.com"

EXPOSE 179  50051

RUN apt-get update && apt-get install -y curl && \
    curl -SL https://github.com/osrg/gobgp/releases/download/v2.10.0/gobgp_2.10.0_linux_amd64.tar.gz | tar xvz -C /usr/local/bin/

# This directory must be mounted as a local volume with '-v `pwd`/gobgp:/etc/gobgp:rw' docker's command line option.
# The host's file at `pwd`/gobgp/gobgp.conf is used as the configuration file for GoBGP.
RUN mkdir /etc/gpbgp
CMD gobgpd -f /etc/gobgp/gobgp.conf -l debug