FROM ubuntu
RUN apt-get update -yy && \
	apt-get install -yy git make curl pkg-config

RUN curl -s https://storage.googleapis.com/golang/go1.7.linux-amd64.tar.gz | tar -C /usr/local -xzf -
ENV GOPATH=/go
ENV PATH=$PATH:/usr/local/go/bin:/go/bin
RUN mkdir -p /go/bin

RUN curl -sL ftp://xmlsoft.org/libxml2/libxml2-2.9.4.tar.gz | tar -xzf - && \
  cd /libxml2-2.9.4 && \
  ./configure \
    --enable-static \
    --disable-shared \
    --without-gnu-ld \
    --with-c14n \
    --without-catalog \
    --without-debug \
    --without-docbook \
    --without-fexceptions \
    --without-ftp \
    --without-history \
    --without-html \
    --without-http \
    --without-iconv	\
    --without-icu \
    --without-iso8859x \
    --without-legacy \
    --without-mem-debug \
    --without-minimum \
    --with-output \
    --without-pattern \
    --with-push \
    --without-python	\
    --without-reader \
    --without-readline \
    --without-regexps \
    --without-run-debug \
    --with-sax1 \
    --without-schemas \
    --without-schematron \
    --without-threads \
    --without-thread-alloc \
    --with-tree \
    --without-valid \
    --without-writer \
    --without-xinclude \
    --without-xpath \
    --with-xptr \
    --without-modules \
    --without-zlib \
    --without-lzma \
    --without-coverage && \
    make install

RUN \
  curl -sL ftp://ftp.openssl.org/source/openssl-1.0.2h.tar.gz | tar -xzf - && \
  cd openssl-1.0.2h && \
  ./config \
    no-shared \
    no-weak-ssl-ciphers \
    no-ssl2 \
    no-ssl3 \
    no-comp \
    no-idea \
    no-dtls \
    no-hw \
    no-threads \
    no-dso && \
  make depend install

RUN curl -sL http://www.aleksey.com/xmlsec/download/xmlsec1-1.2.22.tar.gz | tar -xzf - && \
	cd xmlsec1-1.2.22 && \
	./configure \
		--enable-static \
		--disable-shared \
		--disable-crypto-dl \
		--disable-apps-crypto-dl \
		--enable-static-linking \
		--without-gnu-ld \
		--with-default-crypto=openssl \
		--with-openssl=/usr/local/ssl \
		--with-libxml=/usr/local \
		--without-nss \
		--without-nspr \
		--without-gcrypt \
		--without-gnutls \
		--without-libxslt && \
	make -C src install && \
	make -C include install && \
	make install-pkgconfigDATA

ADD . /go/src/github.com/crewjam/go-xmlsec
WORKDIR /go/src/github.com/crewjam/go-xmlsec
RUN go get github.com/crewjam/errset
RUN go build -tags static -ldflags '-s -extldflags "-static"' -o /bin/xmldsig ./examples/xmldsig.go
RUN ldd /bin/xmldsig || true
RUN /bin/xmldsig --help || true
