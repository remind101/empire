FROM golang:1.4

RUN go get github.com/tools/godep
RUN go get github.com/codegangsta/gin

ENV PORT 8080

CMD ["/go/bin/gin", "--appPort", "8080", "run", "server"]
