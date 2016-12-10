// Package tracer contains Datadog's tracing client. It is used to trace
// requests as they flow across web servers, databases and microservices so
// that developers have visibility into bottlenecks and troublesome
// requests.
//
// Package tracer has two core objects: Tracers and Spans. Spans represent
// a chunk of computation time. They have names, durations, timestamps and
// other metadata. Tracers are used to create hierarchies of spans in a
// request, buffer and submit them to the server.
package tracer
