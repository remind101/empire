package pod

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/coreos/go-etcd/etcd"
	"github.com/remind101/empire/empire/pkg/timex"
)

// Store is used by the ContainerManager implementation to store Templates,
// and Instances. This package does not define an implementation of
// this interface, so this is something that consumers should implement if they
// wish to use the ContainerManager.
type Store interface {
	// CreateTemplate persists the Template.
	CreateTemplate(*Template) error

	// RemoveTemplate removes a Template from the store.
	RemoveTemplate(*Template) error

	// UpdateTemplate updates the Template.
	UpdateTemplate(*Template) error

	// CreateInstance persists an Instance.
	CreateInstance(*Instance) error

	// RemoveInstance removes an Instance from the store.
	RemoveInstance(*Instance) error

	// Templates returns a slice of Templates. A map of tags can be provided to filter
	// by.
	Templates(map[string]string) ([]*Template, error)

	// Template finds a Template by its id.
	Template(string) (*Template, error)

	// Instances returns a slice of Instances for the Template.
	Instances(templateID string) ([]*Instance, error)
}

// NewEtcdStore returns a new etcd backed Store.
func NewEtcdStore(machines []string) Store {
	prefix := "/empire/pods"
	c := etcd.NewClient(machines)

	return &etcdStore{
		Prefix: prefix,
		client: c,
	}
}

// etcdStore is an implementation of the Store interface backed by etcd.
type etcdStore struct {
	// Prefix is a prefix to prefix the keys with.
	Prefix string

	client *etcd.Client
}

func (s *etcdStore) CreateTemplate(t *Template) error {
	return s.set(s.templateKey(t), t)
}

func (s *etcdStore) RemoveTemplate(t *Template) error {
	if err := s.rm(s.templateKey(t)); err != nil {
		return err
	}

	if err := s.rm(s.instancesKey(t)); err != nil {
		return ignoreCode(err, keyNotFound)
	}

	return nil
}

func (s *etcdStore) UpdateTemplate(t *Template) error {
	return s.update(s.templateKey(t), t)
}

func (s *etcdStore) CreateInstance(i *Instance) error {
	i.CreatedAt = timex.Now()
	return s.set(s.instanceKey(i), i)
}

func (s *etcdStore) RemoveInstance(i *Instance) error {
	return s.rm(s.instanceKey(i))
}

func (s *etcdStore) Templates(tags map[string]string) ([]*Template, error) {
	templates, err := s.templates()
	if err != nil {
		return templates, err
	}

	return filterTemplates(templates, tags), nil
}

func (s *etcdStore) Template(id string) (*Template, error) {
	key := s.templateKey(&Template{ID: id})
	resp, err := s.get(key, false)

	if resp == nil {
		return nil, errors.New("key not found")
	}

	var template Template
	if err := json.Unmarshal([]byte(resp.Node.Value), &template); err != nil {
		return nil, err
	}

	return &template, err
}

func (s *etcdStore) Instances(templateID string) ([]*Instance, error) {
	template, err := s.Template(templateID)
	if err != nil {
		return nil, err
	}

	key := s.instancesKey(template)
	recursive := true
	resp, err := s.get(key, recursive)
	if err != nil {
		return nil, err
	}

	root := resp.Node

	var instances []*Instance

	for _, node := range root.Nodes {
		var instance Instance
		if err := json.Unmarshal([]byte(node.Value), &instance); err != nil {
			return instances, err
		}
		instance.Template = template
		instances = append(instances, &instance)
	}

	return instances, nil
}

func (s *etcdStore) templates() ([]*Template, error) {
	recursive := true
	resp, err := s.get(s.templatesKey(), recursive)
	if err != nil {
		// If the /templates key is not found. Return an empty slice of
		// templates.
		return nil, ignoreCode(err, keyNotFound)
	}

	root := resp.Node

	var templates []*Template

	for _, node := range root.Nodes {
		var template Template
		if err := json.Unmarshal([]byte(node.Value), &template); err != nil {
			return templates, err
		}
		templates = append(templates, &template)
	}

	return templates, nil
}

func (s *etcdStore) get(key string, recursive bool) (*etcd.Response, error) {
	return s.client.Get(s.prefix(key), false, recursive)
}

// set sets the key to the JSON encoded value of v.
func (s *etcdStore) set(key string, v interface{}) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}

	_, err = s.client.Set(s.prefix(key), string(raw), 0)
	return err
}

// update updates key with the JSON encoded value of v.
func (s *etcdStore) update(key string, v interface{}) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}

	_, err = s.client.Update(s.prefix(key), string(raw), 0)
	return err
}

// rm removes the key.
func (s *etcdStore) rm(key string) error {
	recursive := true
	_, err := s.client.Delete(s.prefix(key), recursive)
	return err
}

func (s *etcdStore) templatesKey() string {
	return "/templates"
}

func (s *etcdStore) instancesKey(t *Template) string {
	return fmt.Sprintf("/instances/%s", t.ID)
}

func (s *etcdStore) templateKey(t *Template) string {
	return fmt.Sprintf("%s/%s", s.templatesKey(), t.ID)
}

func (s *etcdStore) instanceKey(i *Instance) string {
	return fmt.Sprintf("%s/%d", s.instancesKey(i.Template), i.Instance)
}

// prefix returns the key prefixed with Prefix.
func (s *etcdStore) prefix(key string) string {
	return fmt.Sprintf("%s%s", s.Prefix, key)
}

// NewMemStore returns a new in memory Store.
func NewMemStore() Store {
	return &store{}
}

// store is an implementation of the Store interface that stores everything
// in memory.
type store struct {
	templates []*Template
	instances []*Instance
}

func newFakeStore() *store {
	return &store{}
}

func (s *store) CreateInstance(instance *Instance) error {
	instance.CreatedAt = timex.Now()
	s.instances = append(s.instances, instance)
	return nil
}

func (s *store) RemoveInstance(instance *Instance) error {
	for i, in := range s.instances {
		if in.Template.ID == instance.Template.ID && in.Instance == instance.Instance {
			s.instances = append(s.instances[:i], s.instances[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("no instance: %d", instance.Instance)
}

func (s *store) Instances(templateID string) ([]*Instance, error) {
	var instances []*Instance

	for _, instance := range s.instances {
		if instance.Template.ID == templateID {
			instances = append(instances, instance)
		}
	}

	return instances, nil
}

func (s *store) CreateTemplate(template *Template) error {
	s.templates = append(s.templates, template)
	return nil
}

func (s *store) Template(templateID string) (*Template, error) {
	for _, template := range s.templates {
		if template.ID == templateID {
			return template, nil
		}
	}

	return nil, nil
}

func (s *store) Templates(tags map[string]string) ([]*Template, error) {
	return filterTemplates(s.templates, tags), nil
}

func (s *store) RemoveTemplate(template *Template) error {
	for i, tmpl := range s.templates {
		if tmpl.ID == template.ID {
			s.templates = append(s.templates[:i], s.templates[i+1:]...)
		}
	}

	return nil
}

func (s *store) UpdateTemplate(template *Template) error {
	for _, tmpl := range s.templates {
		if tmpl.ID == template.ID {
			tmpl.Instances = template.Instances
		}
	}

	return nil
}

// filterTemplates reduces templates into only those that have Tags that match all of
// the provided tags.
func filterTemplates(templates []*Template, tags map[string]string) []*Template {
	// Make a copy of the templates slice.
	filtered := templates[:]

	for k, v := range tags {
		var matched []*Template

		for _, tmpl := range filtered {
			if tmpl.Tags[k] == v {
				matched = append(matched, tmpl)
			}
		}

		filtered = matched
	}

	return filtered
}

// The error code that represents a key not found error. See
// http://goo.gl/bPXrtM.
const keyNotFound = 100

// ignoreCode returns a nil error if the ErrorCode is for a specific set of
// codes.
func ignoreCode(err error, codes ...int) error {
	switch err := err.(type) {
	case *etcd.EtcdError:
		for _, code := range codes {
			if err.ErrorCode == code {
				return nil
			}
		}
		return err
	default:
		return err
	}
}
