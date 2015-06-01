package middleware

import (
	"fmt"
	"net/http"

	"github.com/remind101/newrelic"
	"github.com/remind101/pkg/httpx"
	"golang.org/x/net/context"
)

type NewRelicTracer struct {
	handler  httpx.Handler
	tracer   newrelic.TxTracer
	router   *httpx.Router
	createTx func(string, string, newrelic.TxTracer) newrelic.Tx
}

func NewRelicTracing(h httpx.Handler, router *httpx.Router, tracer newrelic.TxTracer) *NewRelicTracer {
	return &NewRelicTracer{h, tracer, router, createTx}
}

func (h *NewRelicTracer) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	path := templatePath(h.router, r)
	txName := fmt.Sprintf("%s %s", r.Method, path)

	tx := h.createTx(txName, r.URL.String(), h.tracer)
	ctx = newrelic.WithTx(ctx, tx)

	tx.Start()
	defer tx.End()

	return h.handler.ServeHTTPContext(ctx, w, r)
}

func templatePath(router *httpx.Router, r *http.Request) string {
	var tpl string

	route, _, _ := router.Handler(r)
	if route != nil {
		tpl = route.GetPathTemplate()
	}

	if tpl == "" {
		tpl = r.URL.Path
	}

	return tpl
}

func createTx(name, url string, tracer newrelic.TxTracer) newrelic.Tx {
	t := newrelic.NewRequestTx(name, url)
	t.Tracer = tracer
	return t
}
