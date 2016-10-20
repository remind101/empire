FROM ubuntu
RUN apt-get update -yy && \
	apt-get install -yy git make curl libxml2-dev libxmlsec1-dev liblzma-dev pkg-config

RUN curl -s https://storage.googleapis.com/golang/go1.7.linux-amd64.tar.gz | tar -C /usr/local -xzf -
ENV GOPATH=/go
ENV PATH=$PATH:/usr/local/go/bin:/go/bin
RUN mkdir -p /go/bin

ADD . /go/src/github.com/crewjam/go-xmlsec
WORKDIR /go/src/github.com/crewjam/go-xmlsec
RUN go get github.com/crewjam/errset
RUN go build -o /bin/xmldsig ./examples/xmldsig.go

# Check our dynamic library dependencies. This will produce output like:
#
#   linux-vdso.so.1 =>  (0x00007ffffa1d3000)
#   libxmlsec1-openssl.so.1 => /usr/lib/libxmlsec1-openssl.so.1 (0x00007f506b9dc000)
#   libxmlsec1.so.1 => /usr/lib/libxmlsec1.so.1 (0x00007f506b77e000)
#   libxml2.so.2 => /usr/lib/x86_64-linux-gnu/libxml2.so.2 (0x00007f506b3c3000)
#   libpthread.so.0 => /lib/x86_64-linux-gnu/libpthread.so.0 (0x00007f506b1a6000)
#   libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6 (0x00007f506addd000)
#   libcrypto.so.1.0.0 => /lib/x86_64-linux-gnu/libcrypto.so.1.0.0 (0x00007f506a981000)
#   libxslt.so.1 => /usr/lib/x86_64-linux-gnu/libxslt.so.1 (0x00007f506a744000)
#   libdl.so.2 => /lib/x86_64-linux-gnu/libdl.so.2 (0x00007f506a540000)
#   libicuuc.so.55 => /usr/lib/x86_64-linux-gnu/libicuuc.so.55 (0x00007f506a1ab000)
#   libz.so.1 => /lib/x86_64-linux-gnu/libz.so.1 (0x00007f5069f91000)
#   liblzma.so.5 => /lib/x86_64-linux-gnu/liblzma.so.5 (0x00007f5069d6f000)
#   libm.so.6 => /lib/x86_64-linux-gnu/libm.so.6 (0x00007f5069a65000)
#   /lib64/ld-linux-x86-64.so.2 (0x000055cfaf030000)
#   libicudata.so.55 => /usr/lib/x86_64-linux-gnu/libicudata.so.55 (0x00007f5067fae000)
#   libstdc++.so.6 => /usr/lib/x86_64-linux-gnu/libstdc++.so.6 (0x00007f5067c2b000)
#   libgcc_s.so.1 => /lib/x86_64-linux-gnu/libgcc_s.so.1 (0x00007f5067a15000)
RUN ldd /bin/xmldsig || true

RUN /bin/xmldsig --help || true
