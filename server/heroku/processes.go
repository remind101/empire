package heroku

import (
	"fmt"
	"net/http"
	"time"

	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/heroku"
	"github.com/remind101/empire/pkg/hijack"
	streamhttp "github.com/remind101/empire/pkg/stream/http"
	"github.com/remind101/pkg/httpx"
	"github.com/remind101/pkg/timex"
	"golang.org/x/net/context"
)

type Dyno heroku.Dyno

func newDyno(task *empire.Task) *Dyno {
	return &Dyno{
		Command:   task.Command.String(),
		Type:      task.Type,
		Name:      task.Name,
		State:     task.State,
		Size:      task.Constraints.String(),
		UpdatedAt: task.UpdatedAt,
	}
}

func newDynos(tasks []*empire.Task) []*Dyno {
	dynos := make([]*Dyno, len(tasks))

	for i := 0; i < len(tasks); i++ {
		dynos[i] = newDyno(tasks[i])
	}

	return dynos
}

func (h *Server) GetProcesses(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	// Retrieve tasks
	js, err := h.Tasks(ctx, a)
	if err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, newDynos(js))
}

type PostProcessForm struct {
	Command string              `json:"command"`
	Attach  bool                `json:"attach"`
	Env     map[string]string   `json:"env"`
	Size    *empire.Constraints `json:"size"`
}

func (h *Server) PostProcess(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var form PostProcessForm

	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	m, err := findMessage(r)
	if err != nil {
		return err
	}

	if err := Decode(r, &form); err != nil {
		return err
	}

	command, err := empire.ParseCommand(form.Command)
	if err != nil {
		return err
	}

	opts := empire.RunOpts{
		User:        UserFromContext(ctx),
		App:         a,
		Command:     command,
		Env:         form.Env,
		Constraints: form.Size,
		Message:     m,
	}

	if form.Attach {
		header := http.Header{}
		header.Set("Content-Type", "application/vnd.empire.raw-stream")
		stream := &hijack.HijackReadWriter{
			Response: w,
			Header:   header,
		}
		defer stream.Close()
		// Prevent the ELB idle connection timeout to close the connection.
		defer close(streamhttp.Heartbeat(stream, 10*time.Second))

		opts.Input = stream
		opts.Output = stream

		if err := h.Run(ctx, opts); err != nil {
			if stream.Hijacked {
				fmt.Fprintf(stream, "%v\r", err)
				return nil
			}
			return err
		}
	} else {
		if err := h.Run(ctx, opts); err != nil {
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

func (h *Server) DeleteProcesses(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	vars := httpx.Vars(ctx)
	pid := vars["pid"]

	if vars["ptype"] != "" {
		return errNotImplemented("Restarting a process type is currently not implemented.")
	}

	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	m, err := findMessage(r)
	if err != nil {
		return err
	}

	if err := h.Restart(ctx, empire.RestartOpts{
		User:    UserFromContext(ctx),
		App:     a,
		PID:     pid,
		Message: m,
	}); err != nil {
		return err
	}

	return NoContent(w)
}
