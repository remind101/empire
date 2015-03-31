package pod

import (
	"testing"

	"github.com/coreos/go-etcd/etcd"
)

func TestStore_CreateTemplate(t *testing.T) {
	s := newTestStore(t)
	defer s.Clean(t)

	if err := s.CreateTemplate(&Template{
		ID: "foo",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestStore_RemoveTemplate(t *testing.T) {
	s := newTestStore(t)
	defer s.Clean(t)

	tmpl := &Template{
		ID: "foo",
	}

	if err := s.RemoveTemplate(tmpl); err == nil {
		t.Fatal("expected an error")
	}

	if err := s.CreateTemplate(tmpl); err != nil {
		t.Fatal(err)
	}

	if err := s.RemoveTemplate(tmpl); err != nil {
		t.Fatal(err)
	}
}

func TestStore_UpdateTemplate(t *testing.T) {
	s := newTestStore(t)
	defer s.Clean(t)

	tmpl := &Template{
		ID: "foo",
	}

	if err := s.UpdateTemplate(tmpl); err == nil {
		t.Fatal("expected an error")
	}

	if err := s.CreateTemplate(tmpl); err != nil {
		t.Fatal(err)
	}

	if err := s.UpdateTemplate(tmpl); err != nil {
		t.Fatal(err)
	}
}

func TestStore_CreateInstance(t *testing.T) {
	s := newTestStore(t)
	defer s.Clean(t)

	tmpl := &Template{
		ID: "foo",
	}
	i := &Instance{
		Template: tmpl,
		Instance: 1,
	}

	if err := s.CreateInstance(i); err != nil {
		t.Fatal(err)
	}
}

func TestStore_RemoveInstance(t *testing.T) {
	s := newTestStore(t)
	defer s.Clean(t)

	tmpl := &Template{
		ID: "foo",
	}
	i := &Instance{
		Template: tmpl,
		Instance: 1,
	}

	if err := s.RemoveInstance(i); err == nil {
		t.Fatal("expected an error")
	}

	if err := s.CreateInstance(i); err != nil {
		t.Fatal(err)
	}

	if err := s.RemoveInstance(i); err != nil {
		t.Fatal(err)
	}
}

func TestStore_Templates(t *testing.T) {
	s := newTestStore(t)
	defer s.Clean(t)

	templates, err := s.Templates(nil)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(templates), 0; got != want {
		t.Fatalf("len(templates) => %d; want %d", got, want)
	}

	s.CreateTemplates(t, []Template{
		Template{ID: "api.web", Tags: map[string]string{"app": "api", "release": "v3"}},
		Template{ID: "shorty.web", Tags: map[string]string{"app": "shorty"}},
	})

	templates, err = s.Templates(nil)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(templates), 2; got != want {
		t.Fatalf("len(templates) => %d; want %d", got, want)
	}

	templates, err = s.Templates(map[string]string{"app": "api"})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(templates), 1; got != want {
		t.Fatalf("len(templates) => %d; want %d", got, want)
	}

	templates, err = s.Templates(map[string]string{"app": "api", "release": "v3"})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(templates), 1; got != want {
		t.Fatalf("len(templates) => %d; want %d", got, want)
	}

	templates, err = s.Templates(map[string]string{"app": "api", "release": "v3"})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(templates), 1; got != want {
		t.Fatalf("len(templates) => %d; want %d", got, want)
	}
}

func TestStore_Template(t *testing.T) {
	s := newTestStore(t)
	defer s.Clean(t)

	tmpl := &Template{
		ID: "shorty.web",
	}

	_, err := s.Template("shorty.web")
	if err == nil {
		t.Fatal("expected an error")
	}

	if err := s.CreateTemplate(tmpl); err != nil {
		t.Fatal(err)
	}

	template, err := s.Template("shorty.web")
	if err != nil {
		t.Fatal(err)
	}

	if template == nil {
		t.Fatal("expected a template")
	}

	if got, want := template.ID, "shorty.web"; got != want {
		t.Fatalf("ID => %s; want %s", got, want)
	}
}

func TestStore_Instances(t *testing.T) {
	s := newTestStore(t)
	defer s.Clean(t)

	tmpl := &Template{
		ID: "shorty.web",
	}

	if err := s.CreateTemplate(tmpl); err != nil {
		t.Fatal(err)
	}

	i := &Instance{
		Template: tmpl,
	}

	instances, err := s.Instances("shorty.web")
	if err == nil {
		t.Fatal("expected an error")
	}

	if got, want := len(instances), 0; got != want {
		t.Fatalf("len(instances) => %d; want %d", got, want)
	}

	if err := s.CreateInstance(i); err != nil {
		t.Fatal(err)
	}

	instances, err = s.Instances("shorty.web")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(instances), 1; got != want {
		t.Fatalf("len(instances) => %d; want %d", got, want)
	}

	if instances[0].Template == nil {
		t.Fatalf("expected template to be set")
	}
}

// etcdTestStore wraps an etcdStore with a function to clean the prefix.
type etcdTestStore struct {
	*etcdStore
}

func newTestStore(t *testing.T) *etcdTestStore {
	c := etcd.NewClient([]string{"http://127.0.0.1:4001"})
	s := &etcdStore{
		Prefix: "/_test",
		client: c,
	}

	return &etcdTestStore{s}
}

func (s *etcdTestStore) Clean(t testing.TB) {
	if _, err := s.client.Delete(s.Prefix, true); err != nil {
		t.Fatal(err)
	}
}

func (s *etcdTestStore) CreateTemplates(t testing.TB, templates []Template) {
	for _, tmpl := range templates {
		if err := s.CreateTemplate(&tmpl); err != nil {
			t.Fatal(err)
		}
	}
}
