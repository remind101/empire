package reporter

import "github.com/jcoene/honeybadger"

// HoneybadgerHandler is a Handler implementation backed for Honeybadger.
type HoneybadgerHandler struct{}

func (h *HoneybadgerHandler) Report(err error) error {
	return honeybadger.Send(err)
}
