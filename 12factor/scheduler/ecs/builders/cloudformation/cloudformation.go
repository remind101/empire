// Package cloudformation provides a StackBuilder for provisioning the AWS
// resources for an App using a CloudFormation stack.
package cloudformation

import (
	"text/template"

	"github.com/remind101/empire/12factor"
)

// StackBuilder is an implementation of the ecs.StackBuilder interface that
// builds the stack using CloudFormation.
type StackBuilder struct {
	// Template is a text/template that will be executed using the App as
	// data. This template should return a valid CloudFormation JSON
	// manifest.
	Template *template.Template
}

// Build builds the CloudFormation stack for the App.
func (b *StackBuilder) Build(app twelvefactor.App) error {
	return nil
}

func (b *StackBuilder) Service(app, process string) (string, error) {
	return "", nil
}
