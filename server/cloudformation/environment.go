package cloudformation

import "fmt"

// EnvironmentProperties are the properties provided to the
// Custom::Environment custom resource.
type EnvironmentProperties struct {
	Environment map[string]string
}

// EnvironmentResource is a custom resource that takes some encrypted
// environment variables, stores them, then returns a unique identifier to
// represent the environment.
type EnvironmentResource struct {
}

func (p *EnvironmentResource) Properties() interface{} {
	return &EnvironmentProperties{}
}

func (p *EnvironmentResource) Provision(req Request) (string, interface{}, error) {
	properties := req.ResourceProperties.(*EnvironmentProperties)

	switch req.RequestType {
	case Create:
		id, err := p.store(properties.Environment)
		return id, nil, err
	case Delete:
		id := req.PhysicalResourceId
		return id, nil, nil
	case Update:
		id, err := p.store(properties.Environment)
		return id, nil, err
	default:
		return "", nil, fmt.Errorf("%s is not supported", req.RequestType)
	}
}

func (p *EnvironmentResource) store(env map[string]string) (string, error) {
	return "env", nil
}
