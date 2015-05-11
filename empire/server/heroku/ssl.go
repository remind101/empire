package heroku

import (
	"net/http"

	"github.com/bgentry/heroku-go"
	"github.com/remind101/empire/empire"
	"github.com/remind101/pkg/httpx"
	"golang.org/x/net/context"
)

type SSLEndpoint heroku.SSLEndpoint

func newSSLEndpoint(cert *empire.Certificate) *SSLEndpoint {
	return &SSLEndpoint{
		Id:               cert.ID,
		Name:             cert.Name,
		CertificateChain: cert.CertificateChain,
		CreatedAt:        *cert.CreatedAt,
		UpdatedAt:        *cert.UpdatedAt,
	}
}

type GetSSLEndpoints struct {
	*empire.Empire
}

func (h *GetSSLEndpoints) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}
	endpoints := make([]*SSLEndpoint, 0)

	cert, err := h.CertificatesFindByApp(ctx, a)
	if err != nil {
		return err
	}

	if cert != nil {
		endpoints = append(endpoints, newSSLEndpoint(cert))
	}

	w.WriteHeader(200)
	return Encode(w, endpoints)
}

type PostSSLEndpointsForm struct {
	CertificateChain string `json:"certificate_chain"`
	Preprocess       bool   `json:"preprocess"`
	PrivateKey       string `json:"private_key"`
}

type PostSSLEndpoints struct {
	*empire.Empire
}

func (h *PostSSLEndpoints) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	var form PostSSLEndpointsForm
	if err := Decode(r, &form); err != nil {
		return err
	}

	cert, err := h.CertificatesCreate(ctx, &empire.Certificate{
		AppID:            a.ID,
		CertificateChain: form.CertificateChain,
		PrivateKey:       form.PrivateKey,
	})
	if err != nil {
		return err
	}

	w.WriteHeader(201)
	return Encode(w, newSSLEndpoint(cert))
}

type PatchSSLEndpoint struct {
	*empire.Empire
}

func (h *PatchSSLEndpoint) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	cert, err := findCert(ctx, a, h)
	if err != nil {
		return err
	}

	var form PostSSLEndpointsForm
	if err := Decode(r, &form); err != nil {
		return err
	}

	cert.CertificateChain = form.CertificateChain
	cert.PrivateKey = form.PrivateKey

	cert, err = h.CertificatesUpdate(ctx, cert)
	if err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, newSSLEndpoint(cert))
}

type DeleteSSLEndpoint struct {
	*empire.Empire
}

func (h *DeleteSSLEndpoint) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	cert, err := findCert(ctx, a, h)
	if err != nil {
		return err
	}

	if err := h.CertificatesDestroy(ctx, cert); err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, newSSLEndpoint(cert))
}

type CertFinder interface {
	CertificatesFind(ctx context.Context, name string) (*empire.Certificate, error)
}

func findCert(ctx context.Context, app *empire.App, f CertFinder) (*empire.Certificate, error) {
	vars := httpx.Vars(ctx)
	name := vars["cert"]

	cert, err := f.CertificatesFind(ctx, name)
	if err != nil {
		return cert, err
	}
	if cert == nil {
		return cert, ErrNotFound
	}

	if app.ID != cert.AppID {
		return cert, ErrNotFound
	}

	return cert, err
}
