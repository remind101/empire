package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"time"
)

type x509Chain []x509.Certificate

func (xc *x509Chain) CommonNames() []string {
	if xc == nil || len(*xc) == 0 {
		return []string{}
	}
	return (*xc)[0].DNSNames
}

func (xc *x509Chain) Expires() time.Time {
	if xc == nil || len(*xc) == 0 {
		return time.Time{}
	}
	return (*xc)[0].NotAfter
}

func decodeCertChain(chainPEM string) (chain x509Chain, err error) {
	certPEMBlock := []byte(chainPEM)
	var certDERBlock *pem.Block
	var cert tls.Certificate

	for {
		certDERBlock, certPEMBlock = pem.Decode([]byte(certPEMBlock))
		if certDERBlock == nil {
			break
		}
		if certDERBlock.Type == "CERTIFICATE" {
			cert.Certificate = append(cert.Certificate, certDERBlock.Bytes)
		}
	}

	if len(cert.Certificate) == 0 {
		err = errors.New("failed to parse certificate PEM data")
		return
	}

	var x509Cert *x509.Certificate
	for _, c := range cert.Certificate {
		x509Cert, err = x509.ParseCertificate(c)
		if err != nil {
			return
		}
		chain = append(chain, *x509Cert)
	}
	return
}
