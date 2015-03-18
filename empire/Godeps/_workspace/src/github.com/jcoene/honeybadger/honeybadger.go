package honeybadger

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"

	_ "crypto/sha512"
)

type Notifier struct {
	Name     string `json:"name"`
	Url      string `json:"url"`
	Version  string `json:"version"`
	Language string `json:"language"`
}

type BacktraceLine struct {
	Method string `json:"method"`
	File   string `json:"file"`
	Number string `json:"number"`
}

type Error struct {
	Class     string                 `json:"class"`
	Message   string                 `json:"message"`
	Backtrace []*BacktraceLine       `json:"backtrace"`
	Source    map[string]interface{} `json:"source"`
}

type Request struct {
	Url       string                 `json:"url"`
	Component string                 `json:"component"`
	Action    string                 `json:"action"`
	Params    map[string]interface{} `json:"params"`
	Session   map[string]interface{} `json:"session"`
	CgiData   map[string]interface{} `json:"cgi_data"`
	Context   map[string]interface{} `json:"context"`
}

type Server struct {
	ProjectRoot     map[string]interface{} `json:"project_root"`
	EnvironmentName string                 `json:"environment_name"`
	Hostname        string                 `json:"hostname"`
}

type Report struct {
	Notifier *Notifier `json:"notifier"`
	Error    *Error    `json:"error"`
	Request  *Request  `json:"request"`
	Server   *Server   `json:"server"`
}

var (
	ApiKey      string
	Environment string
	Client      *http.Client = http.DefaultClient
	reports     chan *Report
)

func Start(apiKey, environment string) {
	ApiKey = apiKey
	Environment = environment

	if ApiKey == "" {
		log.Println("Honeybadger disabled")
		return
	}

	const numWorkers = 10
	const bufferSize = 1024

	reports = make(chan *Report, bufferSize)
	for i := 0; i < numWorkers; i++ {
		go consume()
	}

	log.Println("Honeybadger client started")
}

func consume() {
	for report := range reports {
		if e := report.Send(); e != nil {
			log.Printf("honeybadger: could not send report: %s", e)
		}
	}
}

// Send sends a report to HB synchronously and returns the error
func Send(err error) error {
	if ApiKey == "" {
		return err
	}
	report, e := NewReport(err)
	if e != nil {
		log.Printf("honeybadger: could not create report: %s", e)
		return err
	}
	if e := report.Send(); e != nil {
		log.Printf("honeybadger: could not send report: %s", e)
	}
	return err
}

// Dispatch sends a report to HB asynchronously and returns the error
func Dispatch(err error) error {
	return DispatchWithContext(err, nil)
}

// DispatchWithContext sends a report to HB asynchronously with context and
// returns the error
func DispatchWithContext(err error, context map[string]interface{}) error {
	if ApiKey == "" {
		return err
	}
	report, rErr := NewReport(err)
	if rErr != nil {
		log.Printf("honeybadger: could not create report: %s", rErr)
		return err
	}
	if context != nil {
		for k, v := range context {
			report.AddContext(k, v)
		}
	}

	select {
	case reports <- report:
	default:
		log.Printf("honeybadger: queue is full, dropping on the floor: %s", err)
	}
	return err
}

// Create a new report using the given error message and current call stack.
func NewReport(msg interface{}) (r *Report, err error) {
	return NewReportWithSkipCallers(msg, 0)
}

func fullMessage(msg interface{}) string {
	return fmt.Sprintf("%s", msg)
}

func exceptionClass(message string) string {
	pieces := strings.Split(message, ":")
	for i := len(pieces) - 1; i >= 0; i-- {
		err := strings.TrimSpace(pieces[i])
		if len(err) > 0 {
			return err
		}
	}
	return ""
}

// Create a new report using the given error message and current call stack.
// Supply an integer indicating how many callers to skip (0 is none).
func NewReportWithSkipCallers(msg interface{}, skipCallers int) (r *Report, err error) {
	var cwd, hostname string

	if ApiKey == "" {
		err = errors.New("You must set an API key first")
		return
	}

	if Environment == "" {
		err = errors.New("You must set an environment first")
		return
	}

	cwd, _ = os.Getwd()
	hostname, _ = os.Hostname()

	fullMessage := fullMessage(msg)
	exceptionClass := exceptionClass(fullMessage)

	r = &Report{
		Notifier: &Notifier{
			Name:     "Honeybadger (Go)",
			Url:      "https://github.com/jcoene/honeybadger-go",
			Version:  "1.0",
			Language: "Go",
		},
		Error: &Error{
			Class:     exceptionClass,
			Message:   fullMessage,
			Backtrace: make([]*BacktraceLine, 0),
			Source:    make(map[string]interface{}),
		},
		Request: &Request{
			Params:  make(map[string]interface{}),
			Session: make(map[string]interface{}),
			CgiData: make(map[string]interface{}),
			Context: make(map[string]interface{}),
		},
		Server: &Server{
			ProjectRoot:     make(map[string]interface{}),
			EnvironmentName: Environment,
			Hostname:        hostname,
		},
	}

	callers := make([]uintptr, 10)
	runtime.Callers(0, callers)

	for i, pc := range callers {
		if i < 3 { // skip self-inflicted depth
			continue
		}

		fc := runtime.FuncForPC(pc)
		if fc != nil {
			ms := strings.Split(fc.Name(), "/")
			m := ms[len(ms)-1]
			f, n := fc.FileLine(pc)

			// If we reach our origin depth, use the given function name as error class
			if i == skipCallers+2 {
				r.Error.Class = m
			}

			r.Error.Backtrace = append(r.Error.Backtrace, &BacktraceLine{
				Method: m,
				File:   f,
				Number: fmt.Sprintf("%d", n),
			})
		}
	}

	r.Server.ProjectRoot["path"] = cwd

	return
}

// Add a key and given value to the report as context
func (r *Report) AddContext(k string, v interface{}) {
	r.Request.Context[k] = v
}

// Add a key and given value to the report as parameters
func (r *Report) AddParam(k string, v interface{}) {
	r.Request.Params[k] = v
}

// Add a key and given value to the report as session
func (r *Report) AddSession(k string, v interface{}) {
	r.Request.Session[k] = v
}

// Send the report asynchronously
func (r *Report) Dispatch() {
	go func() {
		if err := r.Send(); err != nil {
			log.Printf("honeybadger: could not send report: %s", err)
		}
	}()
}

// Send the report and return an error if present
func (r *Report) Send() (err error) {
	var req *http.Request
	var resp *http.Response
	var payload []byte

	if payload, err = json.MarshalIndent(r, "", "  "); err != nil {
		return
	}

	if req, err = http.NewRequest("POST", "https://api.honeybadger.io/v1/notices", bytes.NewBuffer(payload)); err != nil {
		return
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-API-Key", ApiKey)

	if resp, err = Client.Do(req); err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 201 {
		err = errors.New(fmt.Sprintf("unable to send: error %d", resp.StatusCode))
	}
	return
}
