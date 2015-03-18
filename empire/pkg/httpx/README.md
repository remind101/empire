# Package httpx

Package httpx provides a layer of convenience over "net/http". Specifically:

1. It's **[context.Context](https://godoc.org/golang.org/x/net/context)** aware.
   This is good for storing request specific parameters, such as a request ids
   and for performing deadlines and cancellations across api boundaries in a
   generic way.
2. `httpx.Handler`'s return an `error` which makes handler implementations feel
   more idiomatic and reduces the chance of accidentally forgetting to return.

The most important part of package httpx is the `Handler` interface, which is
defined as:

```go
type Handler interface {
	ServeHTTPContext(context.Context, http.ResponseWriter, *http.Request) error
}
```

## Usage

```go
r := httpx.NewRouter()
r.Handle("GET", "/", httpx.HandlerFunc(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	io.WriteString(w, `ok`)
	return nil
}

// Adapt the router to the http.Handler interface and insert a
// context.Background().
s := middleware.Background(r)

http.ListenAndServe(":8080", s)
```
