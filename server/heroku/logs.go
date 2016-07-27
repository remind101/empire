package heroku

import (
	"net/http"
	"time"

	streamhttp "github.com/remind101/empire/pkg/stream/http"
	"golang.org/x/net/context"
)

type PostLogsForm struct {
	Duration int64
}

func (h *Server) PostLogs(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	var form PostLogsForm
	// We ignore the EOF error for backwards compatability with the "emp"
	// command that doesn't support providing a duration.
	ignoreEOF := true
	if err := DecodeRequest(r, &form, ignoreEOF); err != nil {
		return err
	}

	rw := streamhttp.StreamingResponseWriter(w)

	// Prevent the ELB idle connection timeout to close the connection.
	defer close(streamhttp.Heartbeat(rw, 10*time.Second))

	err = h.StreamLogs(a, rw, time.Duration(form.Duration))
	if err != nil {
		return err
	}

	return nil
}
