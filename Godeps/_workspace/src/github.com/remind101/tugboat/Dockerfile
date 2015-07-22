FROM alpine:3.1
MAINTAINER Eric Holmes <eric@remind101.com>

COPY ./ /go/src/github.com/remind101/tugboat
COPY ./bin/build /build
RUN /build
WORKDIR /var/run/tugboat

CMD ["/bin/tugboat", "server"]
