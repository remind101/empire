package acl

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPolicy_Match(t *testing.T) {
	tests := []struct {
		policy  Policy
		context Context
		match   bool
	}{
		{
			Policy{
				Action:   []string{"empire:ListFoo"},
				Resource: []string{"*"},
			},
			Context{Action: "empire:ListFoo"},
			true,
		},
		{
			Policy{
				Action:   []string{"empire:ListBar"},
				Resource: []string{"*"},
			},
			Context{Action: "empire:ListFoo"},
			false,
		},
		{
			Policy{
				Action:   []string{"empire:*"},
				Resource: []string{"*"},
			},
			Context{Action: "empire:ListFoo"},
			true,
		},
		{
			Policy{
				Action:   []string{"something:*"},
				Resource: []string{"*"},
			},
			Context{Action: "empire:ListFoo"},
			false,
		},
		{
			Policy{
				Action:   []string{"empire:ListFoo"},
				Resource: []string{"name"},
			},
			Context{Action: "empire:ListFoo", Resource: "name"},
			true,
		},
		{
			Policy{
				Action:   []string{"empire:ListFoo"},
				Resource: []string{"foo"},
			},
			Context{Action: "empire:ListFoo", Resource: "bar"},
			false,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			match := tt.policy.Match(tt.context.Action, tt.context.Resource)
			assert.Equal(t, tt.match, match)
		})
	}
}

func TestPolicy_Allowed(t *testing.T) {
	tests := []struct {
		policy  Policy
		context Context
		allowed bool
	}{
		{
			Policy{
				Effect:   Allow,
				Action:   []string{"empire:ListFoo"},
				Resource: []string{"*"},
			},
			Context{Action: "empire:ListFoo", Resource: "foo"},
			true,
		},
		{
			Policy{
				Effect:   Deny,
				Action:   []string{"empire:ListFoo"},
				Resource: []string{"*"},
			},
			Context{Action: "empire:ListFoo", Resource: "foo"},
			false,
		},
		{
			Policy{
				Effect:   Allow,
				Action:   []string{"empire:ListFoo"},
				Resource: []string{"foo"},
			},
			Context{Action: "empire:ListFoo", Resource: "foo"},
			true,
		},
		{
			Policy{
				Effect:   Allow,
				Action:   []string{"empire:ListFoo"},
				Resource: []string{"bar"},
			},
			Context{Action: "empire:ListFoo", Resource: "foo"},
			false,
		},
		{
			Policy{
				Effect:   Allow,
				Action:   []string{"empire:ListFoo"},
				Resource: []string{"bar"},
			},
			Context{Action: "empire:ListBar", Resource: "bar"},
			false,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			allowed := tt.policy.Allowed(tt.context)
			assert.Equal(t, tt.allowed, allowed)
		})
	}
}

func TestPolicies_Allowed(t *testing.T) {
	tests := []struct {
		policies Policies
		context  Context
		allowed  bool
	}{
		{
			Policies{
				{
					Effect:   Allow,
					Action:   []string{"empire:ListFoo"},
					Resource: []string{"*"},
				},
			},
			Context{Action: "empire:ListFoo", Resource: "foo"},
			true,
		},
		{
			Policies{
				{
					Effect:   Allow,
					Action:   []string{"empire:ListFoo"},
					Resource: []string{"*"},
				},
				{
					Effect:   Deny,
					Action:   []string{"empire:ListFoo"},
					Resource: []string{"*"},
				},
			},
			Context{Action: "empire:ListFoo", Resource: "foo"},
			false,
		},
		{
			Policies{
				{
					Effect:   Deny,
					Action:   []string{"empire:ListFoo"},
					Resource: []string{"*"},
				},
				{
					Effect:   Allow,
					Action:   []string{"empire:ListFoo"},
					Resource: []string{"*"},
				},
			},
			Context{Action: "empire:ListFoo", Resource: "foo"},
			false,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			allowed := tt.policies.Allowed(tt.context)
			assert.Equal(t, tt.allowed, allowed)
		})
	}
}
