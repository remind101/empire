package empire

import (
	"encoding/json"
	"net/http"

	"github.com/docker/docker/pkg/jsonmessage"
	streamhttp "github.com/remind101/empire/pkg/stream/http"
	"github.com/remind101/kinesumer"
)

func (e *Empire) StreamLogs(a *App, w http.ResponseWriter) error {
	rw := streamhttp.StreamingResponseWriter(w)

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
		if err := json.NewEncoder(rw).Encode(&msg); err != nil {
			return err
		}
	}
}
