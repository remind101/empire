FROM gliderlabs/alpine:3.1
MAINTAINER Michael Barrett <mike@remind101.com>

RUN apk-install jq curl

ADD etcd_peers.ash /bin/etcd_peers

CMD ["/bin/etcd_peers"]