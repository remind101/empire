package empire

import (
	"bytes"
	"fmt"
	"net/url"
	"text/template"

	"github.com/remind101/empire/pkg/duo"
	"golang.org/x/net/context"
)

// ActionConfirmer is an interface that can be implemented to confirm that an
// action is allowed.
type ActionConfirmer interface {
	fmt.Stringer

	// Confirm should notify the third party of the action being performed,
	// then block until the action has been confirmed.
	Confirm(ctx context.Context, user *User, action Action, params map[string]string) (bool, error)
}

type ActionConfirmerFunc func(context.Context, *User, Action, map[string]string) (bool, error)

func (f ActionConfirmerFunc) Confirm(ctx context.Context, user *User, action Action, params map[string]string) (bool, error) {
	return f(ctx, user, action, params)
}

func (f ActionConfirmerFunc) String() string {
	return "ActionConfirmerFunc"
}

// DuoActionConfirmer is an ActionConfirmer that will send the user a Duo push
// notification to confirm the action before continuing.
type DuoActionConfirmer struct {
	// A template that will be used to determine the users Duo username. The
	// template will be executed with a User object.
	UsernameTemplate *template.Template

	duo *duo.Client
}

func NewDuoActionConfirmer(key, secret, apiHost string) *DuoActionConfirmer {
	c := duo.New(nil)
	c.Key = key
	c.Secret = secret
	c.Host = apiHost

	return &DuoActionConfirmer{duo: c}
}

func (c *DuoActionConfirmer) Confirm(ctx context.Context, user *User, action Action, params map[string]string) (bool, error) {
	username, err := c.username(user)
	if err != nil {
		return false, err
	}

	pushinfo := url.Values{}
	for k, v := range params {
		pushinfo.Set(k, v)
	}

	q := url.Values{}
	q.Add("username", username)
	q.Add("factor", "push")
	q.Add("device", "auto")
	q.Add("type", action.String())
	q.Add("pushinfo", pushinfo.Encode())

	resp, err := c.duo.Auth(q)
	if err != nil {
		return false, err
	}

	return resp.Response.Result == "allow", nil
}

func (c *DuoActionConfirmer) String() string {
	return "Duo"
}

var defaultDuoUsernameTemplate = template.Must(template.New("username").Parse(`{{.Name}}`))

func (c *DuoActionConfirmer) username(user *User) (string, error) {
	t := c.UsernameTemplate
	if t == nil {
		t = defaultDuoUsernameTemplate
	}

	b := new(bytes.Buffer)
	if err := t.Execute(b, user); err != nil {
		return "", fmt.Errorf("duo confirmation: error finding username: %v", err)
	}

	return b.String(), nil
}
