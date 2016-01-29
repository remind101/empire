package conveyor

import (
	"encoding/json"
	"fmt"
	"net/http"
)

var DefaultTransport = &Transport{}

var DefaultClient = &http.Client{
	Transport: DefaultTransport,
}

// Transport is an http.RoundTripper implementation that decodes the Error
// format that Conveyor returns.
type Transport struct {
	// Transport is the HTTP transport to use when making requests.
	// It will default to http.DefaultTransport if nil.
	Transport http.RoundTripper
}

// Forward CancelRequest to underlying Transport
func (t *Transport) CancelRequest(req *http.Request) {
	type canceler interface {
		CancelRequest(*http.Request)
	}
	tr, ok := t.Transport.(canceler)
	if !ok {
		return
	}
	tr.CancelRequest(req)
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Transport == nil {
		t.Transport = http.DefaultTransport
	}

	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		return nil, err
	}

	if err = checkResponse(resp); err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		return nil, err
	}

	return resp, nil
}

func checkResponse(resp *http.Response) error {
	if resp.StatusCode/100 != 2 { // 200, 201, 202, etc
		var e Error
		err := json.NewDecoder(resp.Body).Decode(&e)
		if err != nil {
			return fmt.Errorf("encountered an error : %s", resp.Status)
		}
		if e.ID == ErrNotFound.ID && e.Message == ErrNotFound.Message {
			return ErrNotFound
		}
		return &e
	}
	return nil
}

func (e *Error) Error() string {
	return e.Message
}
