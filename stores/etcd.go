package stores

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"

	"github.com/coreos/etcd/pkg/transport"
	"github.com/coreos/go-etcd/etcd"
)

type EtcdStore struct {
	client *etcd.Client
	ns     string
}

func NewEtcdStore(ns string) (*EtcdStore, error) {
	client, err := NewEtcdClient()
	if err != nil {
		return nil, err
	}

	return &EtcdStore{client: client, ns: ns}, nil
}

func (s *EtcdStore) Set(k string, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	_, err = s.client.Set(s.key(k), string(b), 0)
	return err
}

func (s *EtcdStore) Get(k string, v interface{}) error {
	res, err := s.client.Get(s.key(k), false, false)
	if err != nil {
		return err
	}

	r := strings.NewReader(res.Node.Value)
	return json.NewDecoder(r).Decode(v)
}

func (s *EtcdStore) List(k string, v interface{}) error {
	t := reflect.TypeOf(v)
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Slice {
		return errors.New("v must be a pointer to a slice")
	}

	sliceValue := reflect.Indirect(reflect.ValueOf(v))

	// Get the element type of the slice
	elemType := reflect.TypeOf(v).Elem().Elem()

	pointerElements := elemType.Kind() == reflect.Ptr
	if pointerElements {
		elemType = elemType.Elem()
	}

	r, err := s.client.Get(s.key(k), false, false)
	if err != nil {
		return err
	}

	for _, n := range r.Node.Nodes {
		elem := reflect.New(elemType)

		if err := json.Unmarshal([]byte(n.Value), elem.Interface()); err != nil {
			return err
		}
		if !pointerElements {
			elem = elem.Elem()
		}
		sliceValue.Set(reflect.Append(sliceValue, elem))
	}

	return nil
}

func (s *EtcdStore) key(k string) string {
	return fmt.Sprintf("%s/%s", s.ns, k)
}

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
