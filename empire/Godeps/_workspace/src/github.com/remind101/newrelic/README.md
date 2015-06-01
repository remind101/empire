## NewRelic Go Agent

A small convenience layer that sits on top of [newrelic-go-agent](https://github.com/paulsmith/newrelic-go-agent), to make
it easy to create transactions for NewRelic in Go.

## Caveats

This is alpha software. It has not been tested in a production environment, or any environment for that matter.

## Installing

You'll need to [install the nr_agent_sdk first](https://docs.newrelic.com/docs/agents/agent-sdk/installation-configuration/installing-agent-sdk).

This package will only work on linux platforms. It is also disabled by default. To enable it, use the build flag `newrelic_enabled`:

```
go build -tags newrelic_enabled ./...
```

## Example Usage

``` go
import "github.com/remind101/newrelic"

func main() {
    tx := newrelic.NewTx("/my/transaction/name")
    tx.Start()
    defer tx.End()

    // Add a segment
    tx.StartGeneric("middleware")
    // Do some middleware stuff...
    tx.EndSegment()
}
```

## Using with an http server

This packages works well as an [httpx middleware](https://github.com/remind101/pkg/blob/master/httpx/middleware/newrelic_tracer.go).

Here is an example using [httpx](https://github.com/remind101/pkg/tree/master/httpx), a context aware http handler.

``` go
    r := httpx.NewRouter()

    r.Handle("/articles", &ArticlesHandler{}).Methods("GET")
    r.Handle("/articles/{id}", &ArticleHandler{}).Methods("GET")

    var h httpx.Handler

    // Add NewRelic tracing.
    h = middleware.NewRelicTracing(r, r, &newrelic.NRTxTracer{})

    // Wrap the route in middleware to add a context.Context.
    h = middleware.BackgroundContext(h)

    http.ListenAndServe(":8080", h)
```

The above example will create web transactions named `GET "/articles"` and `GET "/articles/{id}"`.
