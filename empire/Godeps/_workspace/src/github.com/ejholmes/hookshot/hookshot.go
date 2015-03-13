// Copyright 2014 Eric Holmes.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package hookshot is a router that de-multiplexes and authorizes github webhooks.
package hookshot

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/subtle"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	// HeaderEvent is the name of the header that contains the type of event.
	HeaderEvent = "X-GitHub-Event"

	// HeaderSignature is the name of the header that contains the signature.
	HeaderSignature = "X-Hub-Signature"
)

var (
	// DefaultNotFoundHandler is the default NotFoundHandler for a Router instance.
	DefaultNotFoundHandler = http.HandlerFunc(http.NotFound)

	// DefaultUnauthorizedHandler is the default UnauthorizedHandler for a Router
	// instance, which responds with a 403 status and a plain text body.
	DefaultUnauthorizedHandler = http.HandlerFunc(unauthorized)
)

// Router demultiplexes github hooks.
type Router struct {
	// NotFoundHandler is called when a handler is not found for a given GitHub event.
	// The nil value for NotFoundHandler
	NotFoundHandler http.Handler

	routes routes
}

// NewRouter returns a new Router.
func NewRouter() *Router {
	return &Router{
		routes: make(routes),
	}
}

// Handle maps a github event to an http.Handler.
func (r *Router) Handle(event string, h http.Handler) {
	r.routes[event] = h
}

// HandleFunc maps a github event to an http.HandlerFunc.
func (r *Router) HandleFunc(event string, fn func(http.ResponseWriter, *http.Request)) {
	r.Handle(event, http.HandlerFunc(fn))
}

// ServeHTTP implements the http.Handler interface to route a request to an
// appropriate http.Handler, based on the value of the X-GitHub-Event header.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	event := req.Header.Get(HeaderEvent)

	route := r.routes[event]
	if route == nil {
		r.notFound(w, req)
		return
	}

	route.ServeHTTP(w, req)
}

func (r *Router) notFound(w http.ResponseWriter, req *http.Request) {
	if r.NotFoundHandler == nil {
		r.NotFoundHandler = DefaultNotFoundHandler
	}

	r.NotFoundHandler.ServeHTTP(w, req)
}

// routes maps a github event to an http.Handler.
type routes map[string]http.Handler

// SecretHandler is an http.Handler that will verify the authenticity of the
// request.
type SecretHandler struct {
	// The secret to use to verify the request.
	Secret string

	// SetHeader controls what happens when the X-Hub-Signature header value does
	// not match the calculated signature. Setting this value to true will set
	// the X-Calculated-Signature header in the response.
	//
	// It's recommended that you only enable this for debugging purposes.
	SetHeader bool

	// Handler is the http.Handler that will be called if the request is
	// authorized.
	Handler http.Handler

	// Unauthorized is the http.Handler that will be called if the request
	// is not authorized.
	Unauthorized http.Handler
}

// Authorize wraps an http.Handler to verify the authenticity of the request
// using the provided secret.
func Authorize(h http.Handler, secret string) *SecretHandler {
	return &SecretHandler{Handler: h, Secret: secret}
}

// ServeHTTP implements the http.Handler interface.
func (h *SecretHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if h.Unauthorized == nil {
		h.Unauthorized = DefaultUnauthorizedHandler
	}

	// If a secret is provided, ensure that the request is verified.
	if h.Secret != "" {
		sig, ok := IsAuthorized(req, h.Secret)

		if h.SetHeader {
			w.Header().Set("X-Calculated-Signature", sig)
		}

		if !ok {
			h.Unauthorized.ServeHTTP(w, req)
			return
		}
	}

	h.Handler.ServeHTTP(w, req)
}

// Signature calculates the SHA1 HMAC signature of body, signed by the secret.
//
// When github-services makes a POST request, it includes a SHA1 HMAC signature
// of the request body, signed with the secret provided in the webhook configuration.
// See http://goo.gl/Oe4WwR.
func Signature(body []byte, secret string) string {
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write(body)
	return fmt.Sprintf("%x", mac.Sum(nil))
}

// IsAuthorized checks that the calculated signature for the request matches the provided signature in
// the request headers. Returns the calculated signature, and a boolean value
// indicating whether or not the calculated signature matches the
// X-Hub-Signature value.
func IsAuthorized(r *http.Request, secret string) (string, bool) {
	raw, er := ioutil.ReadAll(r.Body)
	if er != nil {
		return "", false
	}

	// Since we're reading the request from the network, r.Body will return EOF if any
	// downstream http.Handler attempts to read it. We set it to a new io.ReadCloser
	// that will read from the bytes in memory.
	r.Body = ioutil.NopCloser(bytes.NewReader(raw))

	sig := "sha1=" + Signature(raw, secret)
	return sig, compareStrings(r.Header.Get(HeaderSignature), sig)
}

// compareStrings compares two strings in constant time.
func compareStrings(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// unauthorized is the default UnauthorizedHandler.
func unauthorized(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "The provided signature in the "+HeaderSignature+" header does not match.", 403)
}
