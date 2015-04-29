// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// Stacks are the different application execution environments available in the
// Heroku platform.
type Stack struct {
	// when stack was introduced
	CreatedAt time.Time `json:"created_at"`

	// unique identifier of stack
	Id string `json:"id"`

	// unique name of stack
	Name string `json:"name"`

	// availability of this stack: beta, deprecated or public
	State string `json:"state"`

	// when stack was last modified
	UpdatedAt time.Time `json:"updated_at"`
}

// Stack info.
//
// stackIdentity is the unique identifier of the Stack.
func (c *Client) StackInfo(stackIdentity string) (*Stack, error) {
	var stack Stack
	return &stack, c.Get(&stack, "/stacks/"+stackIdentity)
}

// List available stacks.
//
// lr is an optional ListRange that sets the Range options for the paginated
// list of results.
func (c *Client) StackList(lr *ListRange) ([]Stack, error) {
	req, err := c.NewRequest("GET", "/stacks", nil)
	if err != nil {
		return nil, err
	}

	if lr != nil {
		lr.SetHeader(req)
	}

	var stacksRes []Stack
	return stacksRes, c.DoReq(req, &stacksRes)
}
