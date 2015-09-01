package empire

import (
	"encoding/json"
	"io"

	"github.com/docker/docker/pkg/jsonmessage"
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
		msg := jsonmessage.JSONMessage{Status: string(rec.Data())}
		if err := json.NewEncoder(w).Encode(&msg); err != nil {
			return err
		}
	}
}
