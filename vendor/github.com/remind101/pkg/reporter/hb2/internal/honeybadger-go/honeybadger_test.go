package honeybadger

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/mock"
)

var (
	mux           *http.ServeMux
	ts            *httptest.Server
	requests      []*HTTPRequest
	defaultConfig = *Config
)

type MockedHandler struct {
	mock.Mock
}

func (h *MockedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Called()
}

type HTTPRequest struct {
	Request *http.Request
	Body    []byte
}

func (h *HTTPRequest) decodeJSON() hash {
	var dat hash
	err := json.Unmarshal(h.Body, &dat)
	if err != nil {
		panic(err)
	}
	return dat
}

func newHTTPRequest(r *http.Request) *HTTPRequest {
	body, _ := ioutil.ReadAll(r.Body)
	return &HTTPRequest{r, body}
}

func setup(t *testing.T) {
	mux = http.NewServeMux()
	ts = httptest.NewServer(mux)
	requests = []*HTTPRequest{}
	mux.HandleFunc("/v1/notices",
		func(w http.ResponseWriter, r *http.Request) {
			assertMethod(t, r, "POST")
			requests = append(requests, newHTTPRequest(r))
			w.WriteHeader(201)
			fmt.Fprint(w, `{"id":"87ded4b4-63cc-480a-b50c-8abe1376d972"}`)
		},
	)

	*DefaultClient.Config = *newConfig(Configuration{APIKey: "badgers", Endpoint: ts.URL})
}

func teardown() {
	*DefaultClient.Config = defaultConfig
}

func TestDefaultConfig(t *testing.T) {
	if Config.APIKey != "" {
		t.Errorf("Expected Config.APIKey to be empty by default. expected=%#v result=%#v", "", Config.APIKey)
	}
}

func TestConfigure(t *testing.T) {
	Configure(Configuration{APIKey: "badgers"})
	if Config.APIKey != "badgers" {
		t.Errorf("Expected Configure to override config.APIKey. expected=%#v actual=%#v", "badgers", Config.APIKey)
	}
}

func TestNotify(t *testing.T) {
	setup(t)
	defer teardown()

	res, _ := Notify(errors.New("Cobras!"))

	if uuid.Parse(res) == nil {
		t.Errorf("Expected Notify() to return a UUID. actual=%#v", res)
	}

	Flush()

	if !testRequestCount(t, 1) {
		return
	}

	testNoticePayload(t, requests[0].decodeJSON())
}

func TestNotifyWithContext(t *testing.T) {
	setup(t)
	defer teardown()

	context := Context{"foo": "bar"}
	Notify("Cobras!", context)
	Flush()

	if !testRequestCount(t, 1) {
		return
	}

	payload := requests[0].decodeJSON()
	if !testNoticePayload(t, payload) {
		return
	}

	assertContext(t, payload, context)
}

func TestNotifyWithErrorClass(t *testing.T) {
	setup(t)
	defer teardown()

	Notify("Cobras!", ErrorClass{"Badgers"})
	Flush()

	if !testRequestCount(t, 1) {
		return
	}

	payload := requests[0].decodeJSON()
	error_payload, _ := payload["error"].(map[string]interface{})
	sent_klass, _ := error_payload["class"].(string)

	if !testNoticePayload(t, payload) {
		return
	}

	if sent_klass != "Badgers" {
		t.Errorf("Custom error class should override default. expected=%v actual=%#v.", "Badgers", sent_klass)
		return
	}
}

func TestNotifyWithFingerprint(t *testing.T) {
	setup(t)
	defer teardown()

	Notify("Cobras!", Fingerprint{"Badgers"})
	Flush()

	if !testRequestCount(t, 1) {
		return
	}

	payload := requests[0].decodeJSON()
	error_payload, _ := payload["error"].(map[string]interface{})
	sent_fingerprint, _ := error_payload["fingerprint"].(string)

	if !testNoticePayload(t, payload) {
		return
	}

	if sent_fingerprint != "Badgers" {
		t.Errorf("Custom fingerprint should override default. expected=%v actual=%#v.", "Badgers", sent_fingerprint)
		return
	}
}

func TestNotifyWithHandler(t *testing.T) {
	setup(t)
	defer teardown()

	BeforeNotify(func(n *Notice) error {
		n.Fingerprint = "foo bar baz"
		return nil
	})
	Notify(errors.New("Cobras!"))
	Flush()

	payload := requests[0].decodeJSON()
	error_payload, _ := payload["error"].(map[string]interface{})
	sent_fingerprint, _ := error_payload["fingerprint"].(string)

	if !testRequestCount(t, 1) {
		return
	}

	if sent_fingerprint != "foo bar baz" {
		t.Errorf("Handler fingerprint should override default. expected=%v actual=%#v.", "foo bar baz", sent_fingerprint)
		return
	}
}

func TestNotifyWithHandlerError(t *testing.T) {
	setup(t)
	defer teardown()

	err := fmt.Errorf("Skipping this notification")

	BeforeNotify(func(n *Notice) error {
		return err
	})
	_, notifyErr := Notify(errors.New("Cobras!"))
	Flush()

	if !testRequestCount(t, 0) {
		return
	}

	if notifyErr != err {
		t.Errorf("Notify should return error from handler. expected=%v actual=%#v.", err, notifyErr)
		return
	}
}

// Helper functions.

func assertContext(t *testing.T, payload hash, expected Context) {
	var request, context hash
	var ok bool

	request, ok = payload["request"].(map[string]interface{})
	if !ok {
		t.Errorf("Missing request in payload actual=%#v.", payload)
		return
	}

	context, ok = request["context"].(map[string]interface{})
	if !ok {
		t.Errorf("Missing context in request payload actual=%#v.", request)
		return
	}

	for k, v := range expected {
		if context[k] != v {
			t.Errorf("Expected context to include hash. expected=%#v actual=%#v", expected, context)
			return
		}
	}
}

func testRequestCount(t *testing.T, num int) bool {
	if len(requests) != num {
		t.Errorf("Expected %v request to have been made. expected=%#v actual=%#v", num, num, len(requests))
		return false
	}
	return true
}

func testNoticePayload(t *testing.T, payload hash) bool {
	for _, key := range []string{"notifier", "error", "request", "server"} {
		switch payload[key].(type) {
		case map[string]interface{}:
			// OK
		default:
			t.Errorf("Expected payload to include %v hash. expected=%#v actual=%#v", key, key, payload)
			return false
		}
	}
	return true
}

func TestHandlerCallsHandler(t *testing.T) {
	mockHandler := &MockedHandler{}
	mockHandler.On("ServeHTTP").Return()

	handler := Handler(mockHandler)
	req, _ := http.NewRequest("GET", "", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	mockHandler.AssertCalled(t, "ServeHTTP")
}

func TestMetricsHandlerCallsHandler(t *testing.T) {
	mockHandler := &MockedHandler{}
	mockHandler.On("ServeHTTP").Return()

	metricsHandler := MetricsHandler(mockHandler)

	req, _ := http.NewRequest("GET", "", nil)
	w := httptest.NewRecorder()
	metricsHandler.ServeHTTP(w, req)

	mockHandler.AssertCalled(t, "ServeHTTP")
}

func assertMethod(t *testing.T, r *http.Request, method string) {
	if r.Method != method {
		t.Errorf("Unexpected request method. actual=%#v expected=%#v", r.Method, method)
	}
}
