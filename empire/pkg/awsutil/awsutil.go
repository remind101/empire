package awsutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

// Request represents an expected AWS API Operation.
type Request struct {
	RequestURI string
	Operation  string
	Body       string
}

func (r *Request) String() string {
	body := formatBody(strings.NewReader(r.Body))
	return fmt.Sprintf("RequestURI: %s\nOperation: %s\nBody: %s", r.RequestURI, r.Operation, body)
}

// Response represents a predefined response.
type Response struct {
	StatusCode int
	Body       string
}

// Cycle represents a request-response cycle.
type Cycle struct {
	Request  Request
	Response Response
}

// Handler is an http.Handler that will play back cycles.
type Handler struct {
	cycles []Cycle
}

// NewHandler returns a new Handler instance.
func NewHandler(c []Cycle) *Handler {
	return &Handler{cycles: c}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(h.cycles) == 0 {
		fmt.Println("No cycles remaining to replay.")
		w.WriteHeader(404)
		return
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	cycle := h.cycles[0]
	match := Request{
		RequestURI: r.URL.RequestURI(),
		Operation:  r.Header.Get("X-Amz-Target"),
		Body:       string(b),
	}

	if cycle.Request.Body == "ignore" {
		match.Body = cycle.Request.Body
	}

	if cycle.Request.String() == match.String() {
		w.WriteHeader(cycle.Response.StatusCode)
		io.WriteString(w, cycle.Response.Body)
	} else {
		fmt.Println("Request does not match next cycle.")
		fmt.Println(cycle.Request.String())
		fmt.Println(match.String())
		w.WriteHeader(404)
	}

	h.cycles = h.cycles[1:]
}

func formatBody(r io.Reader) string {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}

	s, err := formatJSON(bytes.NewReader(b))
	if err == nil {
		return s
	}

	return string(b)
}

func formatJSON(r io.Reader) (string, error) {
	var body map[string]interface{}
	if err := json.NewDecoder(r).Decode(&body); err != nil {
		return "", err
	}

	raw, err := json.MarshalIndent(&body, "", "  ")
	if err != nil {
		return "", err
	}

	return string(raw), nil
}
