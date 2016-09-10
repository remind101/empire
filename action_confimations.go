package empire

import (
	"context"
	"fmt"
	"net/url"

	"github.com/remind101/empire/pkg/duo"
)

// ActionConfirmer is an interface that can be implemented to confirm that an
// action is allowed.
type ActionConfirmer interface {
	// Confirm should notify the third party of the action being performed,
	// then block until the action has been confirmed.
	Confirm(ctx context.Context, user *User, action string, resource string, params map[string]string) (bool, error)
}

// DuoActionConfirmer is an ActionConfirmer that will send the user a Duo push
// notification to confirm the action before continuing.
type DuoActionConfirmer struct {
	duo *duo.Client
}

func NewDuoActionConfirmer(key, secret, apiHost string) *DuoActionConfirmer {
	c := duo.New(nil)
	c.Key = key
	c.Secret = secret
	c.Host = apiHost

	return &DuoActionConfirmer{duo: c}
}

func (c *DuoActionConfirmer) Confirm(ctx context.Context, user *User, action string, resource string, params map[string]string) (bool, error) {
	q := url.Values{}
	q.Add("username", user.Name)
	q.Add("factor", "push")
	q.Add("device", "auto")
	q.Add("type", action)
	q.Add("pushinfo", fmt.Sprintf("resource=%s", resource))

	resp, err := c.duo.Auth(q)
	if err != nil {
		return false, err
	}

	return resp.Response.Result == "allow", nil
}
