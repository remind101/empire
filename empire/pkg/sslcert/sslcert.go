package sslcert

import "regexp"

type Manager interface {
	// Add adds a new certificate and returns a unique id for the added certificate.
	Add(name string, crt string, key string) (id string, err error)

	// Metadata returns any metadata about the certificate for the given id.
	MetaData(id string) (data map[string]string, err error)

	// Remove removes the certificate for the given id.
	Remove(id string) (err error)
}

// Flags are: multiline mode, dot matches newlines, ungreedy.
var CertPattern = regexp.MustCompile(`(?msU)-----BEGIN CERTIFICATE-----(.+)-----END CERTIFICATE-----`)

func PrimaryCertFromChain(chain string) string {
	matches := CertPattern.FindAllString(chain, -1)
	if len(matches) > 0 {
		return matches[0]
	} else {
		return ""
	}
}
