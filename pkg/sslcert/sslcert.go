package sslcert

import (
	"bytes"
	"encoding/pem"
)

type Manager interface {
	// Add adds a new certificate and returns a unique id for the added certificate.
	Add(name string, crt string, key string) (id string, err error)

	// Metadata returns any metadata about the certificate for the given id.
	MetaData(id string) (data map[string]string, err error)

	// Remove removes the certificate for the given id.
	Remove(id string) (err error)
}

// SplitCertChain takes a complete certificate chain (including the primary cert) and
// returns two strings: the primary cert and the rest of the certificate chain, if any.
func SplitCertChain(chain string) (string, string) {
	block, rest := pem.Decode([]byte(chain))
	if block == nil {
		return "", ""
	}

	var out bytes.Buffer
	if err := pem.Encode(&out, block); err != nil {
		return "", ""
	}

	return out.String(), string(rest)
}
