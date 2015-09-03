package heroku

import (
	"net/http"

	"github.com/remind101/empire"
	streamhttp "github.com/remind101/empire/pkg/stream/http"
	"golang.org/x/net/context"
)

type PostLogs struct {
	*empire.Empire
}

func (h *PostLogs) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/plain; boundary=NL")
	rw := streamhttp.StreamingResponseWriter(w)
	err = h.StreamLogs(a, rw)
	if err != nil {
		return err
	}

	return nil
}
