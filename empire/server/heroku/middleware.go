package heroku

import (
	"github.com/remind101/empire/empire"
	"github.com/remind101/empire/empire/pkg/httpx"
	"github.com/remind101/empire/empire/pkg/reporter"
)

type MiddlewareOpts struct {
	// DisableAuthenticate controls whether the endpoint requires
	// authentication or not. Zero value is to enable the Authentication
	// middleware.
	DisableAuthenticate bool
}

func Middleware(e *empire.Empire, h httpx.Handler, opts *MiddlewareOpts) httpx.Handler {
	if opts == nil {
		opts = &MiddlewareOpts{}
	}

	if !opts.DisableAuthenticate {
		h = Authenticate(e, h)
	}

	m := reporter.NewMiddleware(h, e.Reporter)

	return m
}
