// package hb2 is a Go package for sending errors to Honeybadger
// using the official client library
package hb2

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/remind101/pkg/reporter"
	"github.com/remind101/pkg/reporter/hb2/internal/honeybadger-go"
	"golang.org/x/net/context"
)

// Headers that won't be sent to honeybadger.
var IgnoredHeaders = map[string]struct{}{
	"Authorization": struct{}{},
}

type Config struct {
	ApiKey      string
	Environment string
	Endpoint    string
}

type HbReporter struct {
	client *honeybadger.Client
}

// NewReporter returns a new Reporter instance.
func NewReporter(cfg Config) *HbReporter {
	hbCfg := honeybadger.Configuration{}
	hbCfg.APIKey = cfg.ApiKey
	hbCfg.Env = cfg.Environment
	hbCfg.Endpoint = cfg.Endpoint

	return &HbReporter{honeybadger.New(hbCfg)}
}

// exposes honeybadger config for unit tests
func (r *HbReporter) GetConfig() *honeybadger.Configuration {
	return r.client.Config
}

func makeHoneybadgerFrames(stack errors.StackTrace) []*honeybadger.Frame {
	length := len(stack)
	frames := make([]*honeybadger.Frame, length)
	for index, frame := range stack[:length] {
		frames[index] = &honeybadger.Frame{
			Number: fmt.Sprintf("%d", frame),
			File:   fmt.Sprintf("%s", frame),
			Method: fmt.Sprintf("%n", frame),
		}
	}
	return frames
}

func makeHoneybadgerError(err *reporter.Error) honeybadger.Error {
	cause := err.Cause()
	frames := makeHoneybadgerFrames(err.StackTrace())
	return honeybadger.Error{
		Message: err.Error(),
		Class:   reflect.TypeOf(cause).String(),
		Stack:   frames,
	}
}

// Report reports the error to honeybadger.
func (r *HbReporter) Report(ctx context.Context, err error) error {
	extras := []interface{}{}

	if e, ok := err.(*reporter.Error); ok {
		extras = append(extras, getContextData(e))
		if r := e.Request; r != nil {
			extras = append(extras, honeybadger.Params(r.Form), getRequestData(r), *r.URL)
		}
		err = makeHoneybadgerError(e)
	}

	_, clientErr := r.client.Notify(err, extras...)
	return clientErr
}

func getRequestData(r *http.Request) honeybadger.CGIData {
	cgiData := honeybadger.CGIData{}
	replacer := strings.NewReplacer("-", "_")

	for header, values := range r.Header {
		if _, ok := IgnoredHeaders[header]; ok {
			continue
		}
		key := "HTTP_" + replacer.Replace(strings.ToUpper(header))
		cgiData[key] = strings.Join(values, ",")
	}

	cgiData["REQUEST_METHOD"] = r.Method
	return cgiData
}

func getContextData(err *reporter.Error) honeybadger.Context {
	ctx := honeybadger.Context{}
	for key, value := range err.Context {
		ctx[key] = value
	}
	return ctx
}
