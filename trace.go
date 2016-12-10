package empire

import "github.com/remind101/empire/tracer"

// Tracer used for tracing requests.
var Tracer = tracer.NewTracerTransport(tracer.NewHTTPTransport("http://dockerhost:7777/v0.3/traces"))

func NewRootSpan(name, resource string) *tracer.Span {
	return Tracer.NewRootSpan(name, "empire", resource)
}
