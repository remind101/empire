package heroku

import (
	"net/http"

	"github.com/bgentry/heroku-go"
	"github.com/remind101/empire/empire"
	"golang.org/x/net/context"
)

type SSLEndpoint heroku.SSLEndpoint

type GetSSLEndpoints struct {
	*empire.Empire
}

func (h *GetSSLEndpoints) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// a, err := findApp(ctx, h)
	// if err != nil {
	// 	return err
	// }

	endpoints := []SSLEndpoint{}

	w.WriteHeader(200)
	return Encode(w, endpoints)
}

type PostSSLEndpoints struct {
	*empire.Empire
}

func (h *PostSSLEndpoints) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// a, err := findApp(ctx, h)
	// if err != nil {
	// 	return err
	// }

	endpoint := SSLEndpoint{}

	w.WriteHeader(201)
	return Encode(w, endpoint)
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
