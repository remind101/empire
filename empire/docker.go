package empire

import "github.com/fsouza/go-dockerclient"

// newDockerClient returns a new docker.Client using the given socket and certificate path.
func newDockerClient(socket, certPath string) (*docker.Client, error) {
	if certPath != "" {
		cert := certPath + "/cert.pem"
		key := certPath + "/key.pem"
		ca := certPath + "/ca.pem"
		return docker.NewTLSClient(socket, cert, key, ca)
	}

	return docker.NewClient(socket)
}
