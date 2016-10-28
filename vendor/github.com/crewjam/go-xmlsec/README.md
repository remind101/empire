# go-xmlsec

[![](https://godoc.org/github.com/crewjam/go-xmlsec?status.png)](http://godoc.org/github.com/crewjam/go-xmlsec) [![Build Status](https://travis-ci.org/crewjam/go-xmlsec.svg?branch=master)](https://travis-ci.org/crewjam/go-xmlsec)

A partial wrapper for [xmlsec](https://www.aleksey.com/xmlsec). 

As seems to be the case for many things in the XMLish world, the xmldsig and xmlenc standards are more complex that may be nessesary. This library is as general as I could reasonably make it with an eye towards supporting the parts of the standards that are needed to support a SAML implementation. If there are missing bits you feel you need, please raise an issue or submit a pull request. 

# Examples

## Signing

    key, _ := ioutil.ReadFile("saml.key")
    doc, _ := ioutil.ReadAll(os.Stdin)
    signedDoc, err := Sign(key, doc, SignatureOptions{})
    os.Stdout.Write(signedDoc)

## Verifying

    key, _ := ioutil.ReadFile("saml.crt")
    doc, _ := ioutil.ReadAll(os.Stdin)
    err := xmldsig.Verify(key, doc, SignatureOptions{})
    if err == xmldsig.ErrVerificationFailed {
      os.Exit(1)
    }

## Decrypting

    key, _ := ioutil.ReadFile("saml.key")
    doc, _ := ioutil.ReadAll(os.Stdin)
    plaintextDoc, err := Decrypt(key, doc)
    os.Stdout.Write(plaintextDoc)

## Encrypting

    key, _ := ioutil.ReadFile("saml.crt")
    doc, _ := ioutil.ReadAll(os.Stdin)
    encryptedDoc, err := Encrypt(key, doc, EncryptOptions{})
    os.Stdout.Write(encryptedDoc)

# Install

This package uses cgo to wrap libxmlsec. As such, you'll need libxmlsec headers and a C compiler to make it work. On linux, this might look like:

    $ apt-get install libxml2-dev libxmlsec1-dev pkg-config
    $ go get github.com/crewjam/go-xmlsec

On Mac with homebrew, this might look like:

    $ brew install libxmlsec1 libxml2 pkg-config
    $ go get github.com/crewjam/go-xmlsec

# Static Linking

It may annoy you to grow a depenency on the shared libraries for libxmlsec, libxml2, etc. After some fighting, here is what I made work on Linux to get
a static binary. See also `Dockerfile.build-static` which build the example
program using this method.

## Compile libxml

```
curl -sL ftp://xmlsoft.org/libxml2/libxml2-2.9.4.tar.gz | tar -xzf -
cd /libxml2-2.9.4
./configure --enable-static --disable-shared --without-gnu-ld --with-c14n --without-catalog --without-debug --without-docbook  --without-fexceptions  --without-ftp --without-history --without-html --without-http --without-iconv --without-icu --without-iso8859x --without-legacy --without-mem-debug --without-minimum --with-output --without-pattern --with-push --without-python --without-reader --without-readline --without-regexps --without-run-debug --with-sax1 --without-schemas --without-schematron --without-threads --without-thread-alloc --with-tree --without-valid --without-writer --without-xinclude --without-xpath --with-xptr --without-modules --without-zlib --without-lzma --without-coverage
make install
```

## Compile openssl

```
curl -sL ftp://ftp.openssl.org/source/openssl-1.0.2h.tar.gz | tar -xzf -
cd openssl-1.0.2h
./config no-shared no-weak-ssl-ciphers no-ssl2 no-ssl3 no-comp no-idea no-dtls no-hw no-threads no-dso
make install
```

## Compile libxmlsec

```
curl -sL http://www.aleksey.com/xmlsec/download/xmlsec1-1.2.22.tar.gz | tar -xzf -
./configure --enable-static --disable-shared --disable-crypto-dl --disable-apps-crypto-dl --enable-static-linking --without-gnu-ld       --with-default-crypto=openssl --with-openssl=/usr/local/ssl --with-libxml=/usr/local --without-nss --without-nspr --without-gcrypt --without-gnutls --without-libxslt
make -C src install
make -C include install
make install-pkgconfigDATA
```

## Build with static tag

```
go build -tags static -ldflags '-s -extldflags "-static"' -o /bin/xmldsig-static.bin ./examples/xmldsig.go
```

Running `ldd` on the output should produce `not a dynamic executable`.


