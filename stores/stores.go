package stores

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/coreos/etcd/pkg/transport"
	"github.com/coreos/go-etcd/etcd"
)

func NewEtcdClient() (*etcd.Client, error) {
	builder := &etcdBuilder{}
	builder.GetEndpoints()
	builder.GetTransport()
	builder.BuildClient()

	return builder.client, builder.err
}

type etcdBuilder struct {
	err       error
	endpoints []string
	transport *http.Transport
	client    *etcd.Client
}

func (b *etcdBuilder) GetEndpoints() {
	if b.err != nil {
		return
	}

	peerstr := os.Getenv("ETCDCTL_PEERS")

	// If we still don't have peers, use a default
	if peerstr == "" {
		peerstr = "127.0.0.1:4001"
	}

	eps := strings.Split(peerstr, ",")

	for i, ep := range eps {
		u, err := url.Parse(ep)
		if err != nil {
			b.err = err
			return
		}

		if u.Scheme == "" {
			u.Scheme = "http"
		}

		eps[i] = u.String()
	}
}

func (b *etcdBuilder) GetTransport() {
	if b.err != nil {
		return
	}

	tls := transport.TLSInfo{
		CAFile:   os.Getenv("ETCDCTL_CA_FILE"),
		CertFile: os.Getenv("ETCDCTL_CERT_FILE"),
		KeyFile:  os.Getenv("ETCDCTL_KEY_FILE"),
	}

	b.transport, b.err = transport.NewTransport(tls)
}

func (b *etcdBuilder) BuildClient() {
	if b.err != nil {
		return
	}

	b.client = etcd.NewClient(b.endpoints)
	b.client.SetTransport(b.transport)
	if ok := b.client.SyncCluster(); !ok {
		b.err = errors.New("cannot sync with the cluster using endpoints " + strings.Join(b.endpoints, ", "))
	}
}
