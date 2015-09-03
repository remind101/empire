package empire

import (
	"io"

	"github.com/remind101/kinesumer"
)

func (e *Empire) StreamLogs(a *App, w io.Writer) error {
	k, err := kinesumer.NewDefault(a.ID)
	if err != nil {
		return err
	}

	_, err = k.Begin()
	if err != nil {
		return err
	}
	defer k.End()

	for {
		rec := <-k.Records()
		msg := append(rec.Data(), '\n')
		if _, err := w.Write(msg); err != nil {
			return err
		}
	}
}
