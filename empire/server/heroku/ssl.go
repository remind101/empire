package heroku

import (
	"net/http"

	"github.com/bgentry/heroku-go"
	"github.com/remind101/empire/empire"
	"golang.org/x/net/context"
)

type SSLEndpoint heroku.SSLEndpoint

func newSSLEndpoint(cert *empire.Certificate) *SSLEndpoint {
	return &SSLEndpoint{
		Id:               cert.ID,
		Name:             cert.Name,
		CertificateChain: cert.CertificateChain,
		CreatedAt:        cert.CreatedAt,
		UpdatedAt:        cert.UpdatedAt,
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

	cert, err := h.CertificatesFindByApp(ctx, a.Name)
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
		AppID:            a.Name,
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
	// a, err := findApp(ctx, h)
	// if err != nil {
	// 	return err
	// }

	endpoint := SSLEndpoint{}

	w.WriteHeader(200)
	return Encode(w, endpoint)
}

type DeleteSSLEndpoint struct {
	*empire.Empire
}

func (h *DeleteSSLEndpoint) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// a, err := findApp(ctx, h)
	// if err != nil {
	// 	return err
	// }

	endpoint := SSLEndpoint{}

	w.WriteHeader(200)
	return Encode(w, endpoint)
}
