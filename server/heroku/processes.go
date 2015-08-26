package heroku

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bgentry/heroku-go"
	"github.com/remind101/empire"
	streamhttp "github.com/remind101/empire/pkg/stream/http"
	"github.com/remind101/pkg/httpx"
	"github.com/remind101/pkg/timex"
	"golang.org/x/net/context"
)

type Dyno heroku.Dyno

func newDyno(j *empire.ProcessState) *Dyno {
	return &Dyno{
		Command:   j.Command,
		Type:      j.Type,
		Name:      j.Name,
		State:     j.State,
		Size:      j.Constraints.String(),
		UpdatedAt: j.UpdatedAt,
	}
}

func newDynos(js []*empire.ProcessState) []*Dyno {
	dynos := make([]*Dyno, len(js))

	for i := 0; i < len(js); i++ {
		dynos[i] = newDyno(js[i])
	}

	return dynos
}

type GetProcesses struct {
	*empire.Empire
}

func (h *GetProcesses) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	// Retrieve job states
	js, err := h.JobStatesByApp(ctx, a)
	if err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, newDynos(js))
}

type PostProcessForm struct {
	Command string            `json:"command"`
	Attach  bool              `json:"attach"`
	Env     map[string]string `json:"env"`
	Size    string            `json:"size"`
}

type PostProcess struct {
	*empire.Empire
}

func (h *PostProcess) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var form PostProcessForm

	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	if err := Decode(r, &form); err != nil {
		return err
	}

	opts := empire.ProcessRunOpts{
		Command: form.Command,
		Env:     form.Env,
	}

	if form.Attach {
		inStream, outStream, err := hijackServer(w)
		if err != nil {
			return err
		}
		defer closeStreams(inStream, outStream)

		fmt.Fprintf(outStream, "HTTP/1.1 200 OK\r\nContent-Type: application/vnd.empire.raw-stream\r\n\r\n")

		// Prevent the ELB idle connection timeout to close the connection.
		defer close(streamhttp.Heartbeat(outStream, 10*time.Second))

		opts.Input = inStream
		opts.Output = outStream

		if err := h.ProcessesRun(ctx, a, opts); err != nil {
			fmt.Fprintf(outStream, "%v", err)
			return nil
		}
	} else {
		if err := h.ProcessesRun(ctx, a, opts); err != nil {
			return err
		}

		dyno := &heroku.Dyno{
			Name:      "run",
			Command:   form.Command,
			CreatedAt: timex.Now(),
		}

		w.WriteHeader(201)
		return Encode(w, dyno)
	}

	return nil
}

type DeleteProcesses struct {
	*empire.Empire
}

func (h *DeleteProcesses) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	vars := httpx.Vars(ctx)
	pid := vars["pid"]

	if vars["ptype"] != "" {
		return errNotImplemented("Restarting a process type is currently not implemented.")
	}

	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	err = h.ProcessesRestart(ctx, a, pid)
	if err != nil {
		return err
	}

	return NoContent(w)
}

func closeStreams(streams ...interface{}) {
	for _, stream := range streams {
		if tcpc, ok := stream.(interface {
			CloseWrite() error
		}); ok {
			tcpc.CloseWrite()
		} else if closer, ok := stream.(io.Closer); ok {
			closer.Close()
		}
	}
}

func hijackServer(w http.ResponseWriter) (io.ReadCloser, io.Writer, error) {
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		return nil, nil, err
	}
	// Flush the options to make sure the client sets the raw mode
	conn.Write([]byte{})
	return conn, conn, nil
}
