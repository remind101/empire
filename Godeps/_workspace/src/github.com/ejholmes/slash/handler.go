package slash

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"

	"golang.org/x/net/context"
)

var (
	// ErrNoHandler is returned by Mux ServeCommand if a Handler isn't found
	// for the route.
	ErrNoHandler = errors.New("slash: no handler")

	// ErrInvalidToken is returned when the provided token in the request
	// does not match the expected secret.
	ErrInvalidToken = errors.New("slash: invalid token")
)

// Responder represents an object that can send Responses.
type Responder interface {
	Respond(Response) error
}

// Handler represents something that handles a slash command.
type Handler interface {
	// ServeCommand runs the command. The provided Responder object can be
	// used to send responses back to the user. If an error is returned, the
	// string error will be sent back to the user as a response.
	ServeCommand(context.Context, Responder, Command) error
}

// HandlerFunc is a function that implements the Handler interface.
type HandlerFunc func(context.Context, Responder, Command) error

func (fn HandlerFunc) ServeCommand(ctx context.Context, r Responder, command Command) error {
	return fn(ctx, r, command)
}

// Matcher is something that can check if a Command matches a Route.
type Matcher interface {
	Match(Command) (map[string]string, bool)
}

// MatcherFunc is a function that implements Matcher.
type MatcherFunc func(Command) (map[string]string, bool)

func (fn MatcherFunc) Match(command Command) (map[string]string, bool) {
	return fn(command)
}

// MatchCommand returns a Matcher that checks that the command strings match.
func MatchCommand(cmd string) Matcher {
	return MatcherFunc(func(command Command) (map[string]string, bool) {
		return make(map[string]string), command.Command == cmd
	})
}

// MatchSubcommand returns a Matcher that checks for the first string of the
// text portion of a command, assuming it's a subcommand.
func MatchSubcommand(subcmd string) Matcher {
	re := regexp.MustCompile(fmt.Sprintf("^%s.*$", subcmd))
	return MatchTextRegexp(re)
}

// MatchTextRegexp returns a Matcher that checks that the command text matches a
// regular expression.
func MatchTextRegexp(r *regexp.Regexp) Matcher {
	return MatcherFunc(func(command Command) (map[string]string, bool) {
		params := make(map[string]string)
		matches := r.FindStringSubmatch(command.Text)
		if len(matches) == 0 {
			return params, false
		}

		for i, m := range matches {
			k := r.SubexpNames()[i]
			if k != "" {
				params[k] = m
			}
		}

		return params, true
	})
}

// Route wraps a Handler with a Matcher.
type Route struct {
	Handler
	Matcher
}

// NewRoute returns a new Route instance.
func NewRoute(handler Handler) *Route {
	return &Route{
		Handler: handler,
	}
}

// Mux is a Handler implementation that routes commands to Handlers.
type Mux struct {
	routes []*Route
}

// NewMux returns a new Mux instance.
func NewMux() *Mux {
	return &Mux{}
}

// Handle adds a Handler to handle the given command.
//
// Example
//
//	m.Handle("/deploy", "token", DeployHandler)
func (m *Mux) Command(command, token string, handler Handler) *Route {
	return m.Match(MatchCommand(command), ValidateToken(handler, token))
}

// MatchText adds a route that matches when the text of the command matches the
// given regular expression. If the route matches and is called, slash.Matches
// will return the capture groups.
func (m *Mux) MatchText(re *regexp.Regexp, handler Handler) *Route {
	return m.Match(MatchTextRegexp(re), handler)
}

// Match adds a new route that uses the given Matcher to match.
func (m *Mux) Match(matcher Matcher, handler Handler) *Route {
	r := NewRoute(handler)
	r.Matcher = matcher
	return m.addRoute(r)
}

func (m *Mux) addRoute(r *Route) *Route {
	m.routes = append(m.routes, r)
	return r
}

// Handler returns the Handler that can handle the given slash command. If no
// handler matches, nil is returned.
func (m *Mux) Handler(command Command) (Handler, map[string]string) {
	for _, r := range m.routes {
		if params, ok := r.Match(command); ok {
			return r.Handler, params
		}
	}
	return nil, nil
}

// ServeCommand attempts to find a Handler to serve the Command. If no handler
// is found, an error is returned.
func (m *Mux) ServeCommand(ctx context.Context, r Responder, command Command) error {
	h, params := m.Handler(command)
	if h == nil {
		return ErrNoHandler
	}
	return h.ServeCommand(WithParams(ctx, params), r, command)
}

// ValidateToken returns a new Handler that verifies that the token in the
// request matches the given token.
func ValidateToken(h Handler, token string) Handler {
	return HandlerFunc(func(ctx context.Context, r Responder, command Command) error {
		// If an empty string was provided, this was probably a
		// configuration error, so return unauthorized for safety.
		if token == "" {
			return ErrInvalidToken
		}

		if command.Token != token {
			return ErrInvalidToken
		}
		return h.ServeCommand(ctx, r, command)
	})
}

// responder is an implementation of the Responder interface that POST's the
// response to the given url.
type responder struct {
	responseURL *url.URL
	client      *http.Client
}

func newResponder(command Command) *responder {
	return &responder{
		responseURL: command.ResponseURL,
		client:      http.DefaultClient,
	}
}

func (r *responder) Respond(resp Response) error {
	raw, err := json.Marshal(newResponse(resp))
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", r.responseURL.String(), bytes.NewReader(raw))
	if err != nil {
		return err
	}

	hresp, err := r.client.Do(req)
	if err != nil {
		return err
	}

	if hresp.StatusCode/100 != 2 {
		raw, _ := ioutil.ReadAll(hresp.Body)
		return fmt.Errorf("error sending delayed response: %s", raw)
	}

	return err
}

type response struct {
	ResponseType *string `json:"response_type,omitempty"`
	Text         string  `json:"text"`
}

func newResponse(resp Response) *response {
	r := &response{Text: resp.Text}
	if resp.InChannel {
		t := "in_channel"
		r.ResponseType = &t
	}
	return r
}
