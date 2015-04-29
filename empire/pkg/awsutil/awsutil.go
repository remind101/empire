package awsutil

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

// Request represents an expected AWS API Operation.
type Request struct {
	Operation string
	Body      string
}

func (r *Request) String() string {
	body, err := formatJSON(strings.NewReader(r.Body))
	if err != nil {
		body = r.Body
	}
	return fmt.Sprintf("Operation: %s\nBody: %s", r.Operation, body)
}

// Response represents a predefined response.
type Response struct {
	StatusCode int
	Body       string
}

// Handler is an http.Handler that will play back scenarios.
type Handler struct {
	scenarios map[string]Response
}

// NewHandler returns a new Handler instance.
func NewHandler(m map[Request]Response) *Handler {
	s := make(map[string]Response)

	for req, res := range m {
		s[req.String()] = res
	}

	return &Handler{
		scenarios: s,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	body, err := formatJSON(r.Body)
	if err != nil {
		body = string(raw)
	}

	match := Request{
		Operation: r.Header.Get("X-Amz-Target"),
		Body:      body,
	}

	if res, ok := h.scenarios[match.String()]; ok {
		w.WriteHeader(res.StatusCode)
		io.WriteString(w, res.Body)
	} else {
		fmt.Println(match.String())
		w.WriteHeader(404)
	}
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
