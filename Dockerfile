FROM golang:1.24.12-trixie AS builder
LABEL eu.pgpkeys.wkd-proxy.temp=true

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update -qq && \
    apt-get -y upgrade && \
    adduser builder --system --disabled-login && \
    apt-get -y install build-essential --no-install-recommends && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

COPY --chown=builder:root Makefile /wkd-proxy/
COPY --chown=builder:root src /wkd-proxy/src
ENV GOPATH=/wkd-proxy
USER builder
WORKDIR /wkd-proxy
RUN make test
COPY --chown=builder:root .git /wkd-proxy/.git
RUN make build


FROM debian:trixie-slim

ENV DEBIAN_FRONTEND=noninteractive

RUN mkdir -p /wkd-proxy/bin /wkd-proxy/etc && \
    apt-get update -qq && \
    apt-get -y upgrade && \
    apt-get -y install ca-certificates && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*
COPY --from=builder /wkd-proxy/bin /wkd-proxy/bin
COPY bin/startup.sh /wkd-proxy/bin/
COPY etc/wkd-proxy.conf /wkd-proxy/etc/
CMD ["/wkd-proxy/bin/startup.sh"]
