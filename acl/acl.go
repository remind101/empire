// Package acl provides a Go library for performing Access Control using
// policy documents, similar to AWS IAM.
//
// In general, this package follows the same guidelines set forth by IAM when
// evaluating IAM policies. See http://goo.gl/MY8C2r.
package acl

import (
	"context"
	"errors"
)

type Effect int

const (
	Deny Effect = iota
	Allow
)

// ACL is a composition of a Context, and the set of policy documents that
// should be used to determine if the request should be allowed.
type ACL struct {
	Context Context
	Policy  Policy
}

func (l *ACL) Allowed() bool {
	return l.Policy.Allowed(l.Context)
}

// Context represents a compiled request context, which is provided to policies
// to check if the request should be allowed.
type Context struct {
	// The Action that's being invoked.
	Action string

	// The Resource that the action is being invoked on.
	Resource string
}

// Statement is used to define whether an action is allowed or denied.
type Statement struct {
	// Either Allow or Deny. The zero value is Deny.
	Effect Effect

	// Action or list of Actions that is Allowed/Denied. It is a runtime
	// error for this to be empty.
	Action []string

	// Resource or list of Resources that the Actions are Allowed/Denied on.
	// It is a runtime error for this to be empty.
	Resource []string
}

// Checks whether the policy is valid.
func (p *Statement) Valid() error {
	if p.Resource == nil || len(p.Resource) == 0 {
		return errors.New("policy: No resources defined")
	}

	if p.Action == nil || len(p.Action) == 0 {
		return errors.New("policy: No actions defined")
	}

	return nil
}

// Whether this directly applies to the given action.
func (p *Statement) Match(action, resource string) bool {
	for _, a := range p.Action {
		if matchAction(action, a) {
			for _, r := range p.Resource {
				if matchResource(resource, r) {
					return true
				}
			}
		}
	}

	return false
}

// Allowed returns true if this policy allows the action on the given resource.
// Match should be called before this to check if this policy defines an effect
// for the given action.
func (p *Statement) Allowed(context Context) bool {
	if p.Match(context.Action, context.Resource) {
		return p.Effect == Allow
	}

	return false
}

// Policy wraps multiple Statement objects as one, providing a single `Allowed`
// method to check if the action is allowed.
type Policy []Statement

// Allowed returns true if the action is allowed on the resource. Like IAM,
// explicit Denies take precedent. The ordering of the Policy does not matter.
//
// See http://goo.gl/oNQy9m
func (p Policy) Allowed(context Context) bool {
	// By default, everything is denied.
	allowed := false

	for _, policy := range p {
		if policy.Match(context.Action, context.Resource) {
			allowed = policy.Allowed(context)

			// Explicit denies take precedent.
			if !allowed {
				return false
			}
		}
	}

	return allowed
}

func matchAction(action string, matcher string) bool {
	return stringMatch(action, matcher)
}

func matchResource(resource string, matcher string) bool {
	return stringMatch(resource, matcher)
}

func stringMatch(actual string, matcher string) bool {
	if matcher == "*" {
		return true
	}

	return actual == matcher
}

// key used to store context values from within this package.
type key int

const (
	policyKey key = iota
)

// WithPolicy embeds the given acl policies in the context.
func WithPolicy(ctx context.Context, policy Policy) context.Context {
	return context.WithValue(ctx, policyKey, policy)
}

// PolicyFromContext returns the embeded acl policy.
func PolicyFromContext(ctx context.Context) Policy {
	p, ok := ctx.Value(policyKey).(Policy)
	if ok {
		return p
	}
	return Policy{}
}
