package heroku

import (
	"fmt"
	"net/http"
	"time"

	"github.com/remind101/empire"
	streamhttp "github.com/remind101/empire/pkg/stream/http"
	"golang.org/x/net/context"
)

type PostLogs struct {
	*empire.Empire
}

type PostLogsForm struct {
	Duration int64
}

func (h *PostLogs) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(ctx, h)
	if err != nil {
		return fmt.Errorf("error finding app: %v", err)
	}

	var form PostLogsForm
	if err := Decode(r, &form); err != nil {
		if err.Error() != "EOF" {
			return fmt.Errorf("error decoding request: %v", err)
		}
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
