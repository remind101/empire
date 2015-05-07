package sslcert

type Manager interface {
	// Add adds a new certificate and returns a unique id for the added certificate.
	Add(name string, crt string, key string) (id string, err error)

	// Metadata returns any metadata about the certificate for the given id.
	MetaData(id string) (data map[string]string, err error)

	// Remove removes the certificate for the given id.
	Remove(id string) (err error)
}
