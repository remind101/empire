package acl

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatement_Match(t *testing.T) {
	tests := []struct {
		statement Statement
		context   Context
		match     bool
	}{
		{
			Statement{
				Action:   []string{"ListFoo"},
				Resource: []string{"*"},
			},
			Context{Action: "ListFoo"},
			true,
		},
		{
			Statement{
				Action:   []string{"ListBar"},
				Resource: []string{"*"},
			},
			Context{Action: "ListFoo"},
			false,
		},
		{
			Statement{
				Action:   []string{"*"},
				Resource: []string{"*"},
			},
			Context{Action: "ListFoo"},
			true,
		},
		{
			Statement{
				Action:   []string{"something:*"},
				Resource: []string{"*"},
			},
			Context{Action: "ListFoo"},
			false,
		},
		{
			Statement{
				Action:   []string{"ListFoo"},
				Resource: []string{"name"},
			},
			Context{Action: "ListFoo", Resource: "name"},
			true,
		},
		{
			Statement{
				Action:   []string{"ListFoo"},
				Resource: []string{"foo"},
			},
			Context{Action: "ListFoo", Resource: "bar"},
			false,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			match := tt.statement.Match(tt.context.Action, tt.context.Resource)
			assert.Equal(t, tt.match, match)
		})
	}
}

func TestStatement_Allowed(t *testing.T) {
	tests := []struct {
		statement Statement
		context   Context
		allowed   bool
	}{
		{
			Statement{
				Effect:   Allow,
				Action:   []string{"ListFoo"},
				Resource: []string{"*"},
			},
			Context{Action: "ListFoo", Resource: "foo"},
			true,
		},
		{
			Statement{
				Effect:   Deny,
				Action:   []string{"ListFoo"},
				Resource: []string{"*"},
			},
			Context{Action: "ListFoo", Resource: "foo"},
			false,
		},
		{
			Statement{
				Effect:   Allow,
				Action:   []string{"ListFoo"},
				Resource: []string{"foo"},
			},
			Context{Action: "ListFoo", Resource: "foo"},
			true,
		},
		{
			Statement{
				Effect:   Allow,
				Action:   []string{"ListFoo"},
				Resource: []string{"bar"},
			},
			Context{Action: "ListFoo", Resource: "foo"},
			false,
		},
		{
			Statement{
				Effect:   Allow,
				Action:   []string{"ListFoo"},
				Resource: []string{"bar"},
			},
			Context{Action: "ListBar", Resource: "bar"},
			false,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			allowed := tt.statement.Allowed(tt.context)
			assert.Equal(t, tt.allowed, allowed)
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
				{
					Effect:   Allow,
					Action:   []string{"ListFoo"},
					Resource: []string{"*"},
				},
			},
			Context{Action: "ListFoo", Resource: "foo"},
			true,
		},
		{
			Policy{
				{
					Effect:   Allow,
					Action:   []string{"ListFoo"},
					Resource: []string{"*"},
				},
				{
					Effect:   Deny,
					Action:   []string{"ListFoo"},
					Resource: []string{"*"},
				},
			},
			Context{Action: "ListFoo", Resource: "foo"},
			false,
		},
		{
			Policy{
				{
					Effect:   Deny,
					Action:   []string{"ListFoo"},
					Resource: []string{"*"},
				},
				{
					Effect:   Allow,
					Action:   []string{"ListFoo"},
					Resource: []string{"*"},
				},
			},
			Context{Action: "ListFoo", Resource: "foo"},
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
