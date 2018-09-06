package heroku

import (
	"fmt"
	"net/http"

	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/heroku"
	"github.com/remind101/empire/pkg/hijack"
	"github.com/remind101/empire/pkg/stdcopy"
	"github.com/remind101/empire/pkg/timex"
	"github.com/remind101/empire/server/auth"
)

type Dyno heroku.Dyno

func newDyno(task *empire.Task) *Dyno {
	return &Dyno{
		Command:   task.Command.String(),
		Type:      task.Type,
		Name:      task.Name,
		Host:      heroku.Host{Id: task.Host.ID},
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

func (h *Server) GetProcesses(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	a, err := h.findApp(r)
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

func (h *Server) PostProcess(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	var form PostProcessForm

	a, err := h.findApp(r)
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
		User:        auth.UserFromContext(ctx),
		App:         a,
		Command:     command,
		Env:         form.Env,
		Constraints: form.Size,
		Message:     m,
	}

	if form.Attach {
		multiplex := r.Header.Get("X-Multiplex") != ""

		header := http.Header{}
		if multiplex {
			header.Set("Content-Type", "application/vnd.empire.stdcopy-stream")
		} else {
			header.Set("Content-Type", "application/vnd.empire.raw-stream")
		}

		stream := &hijack.HijackReadWriter{
			Response: w,
			Header:   header,
		}
		defer stream.Close()

		stdio := &empire.IO{
			Stdin: stream,
		}

		if multiplex {
			stdio.Stdout = stdcopy.NewStdWriter(stream, stdcopy.Stdout)
			stdio.Stderr = stdcopy.NewStdWriter(stream, stdcopy.Stderr)
		} else {
			// Backwards compatibility for older clients that don't
			// know how to de-multiplex a stdcopy stream. For these
			// clients, stdout/stderr are merged together.
			stdio.Stdout = stream
			stdio.Stderr = stream
		}

		opts.IO = stdio

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

func (h *Server) DeleteProcesses(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	vars := Vars(r)
	pid := vars["pid"]

	if vars["ptype"] != "" {
		return errNotImplemented("Restarting a process type is currently not implemented.")
	}

	a, err := h.findApp(r)
	if err != nil {
		return err
	}

	m, err := findMessage(r)
	if err != nil {
		return err
	}

	if err := h.Restart(ctx, empire.RestartOpts{
		User:    auth.UserFromContext(ctx),
		App:     a,
		PID:     pid,
		Message: m,
	}); err != nil {
		return err
	}

	return NoContent(w)
}
