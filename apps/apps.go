package apps

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/coreos/go-etcd/etcd"
	"github.com/remind101/empire/repos"
	"github.com/remind101/empire/stores"
)

var ErrInvalidName = errors.New("An app name must alphanumeric and dashes only, 3-30 chars in length.")

var NamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{2,30}$`)

// Name represents the unique name for an App.
type Name string

// NewNameFromRepo generates a new name from a Repo
//
//	remind101/r101-api => r101-api
func NewNameFromRepo(repo repos.Repo) Name {
	p := strings.Split(string(repo), "/")
	return Name(p[len(p)-1])
}

// App represents an app.
type App struct {
	Name Name `json:"name"`

	// The associated GitHub/Docker repo.
	Repo repos.Repo `json:"repo"`
}

// New validates the name of the new App then returns a new App instance. If the
// name is invalid, an error is retuend.
func New(name Name, repo repos.Repo) (*App, error) {
	if !NamePattern.Match([]byte(name)) {
		return nil, ErrInvalidName
	}

	return &App{
		Name: name,
		Repo: repo,
	}, nil
}

// NewFromRepo returns a new App initialized from the name of a Repo.
func NewFromRepo(repo repos.Repo) (*App, error) {
	name := NewNameFromRepo(repo)
	return New(name, repo)
}

// Repository represents a repository for creating and finding Apps.
type Repository interface {
	Create(*App) (*App, error)
	FindByName(Name) (*App, error)
	FindByRepo(repos.Repo) (*App, error)
}

func NewRepository() Repository {
	return newRepository()
}

type repository struct {
	id int

	sync.RWMutex
	apps []*App
}

func newRepository() *repository {
	return &repository{apps: make([]*App, 0)}
}

func (r *repository) Create(app *App) (*App, error) {
	r.Lock()
	defer r.Unlock()

	r.apps = append(r.apps, app)
	return app, nil
}

func (r *repository) FindByName(name Name) (*App, error) {
	r.RLock()
	defer r.RUnlock()

	for _, app := range r.apps {
		if app.Name == name {
			return app, nil
		}
	}

	return nil, nil
}

func (r *repository) FindByRepo(repo repos.Repo) (*App, error) {
	r.RLock()
	defer r.RUnlock()

	for _, app := range r.apps {
		if app.Repo == repo {
			return app, nil
		}
	}

	return nil, nil
}

type etcdRepo struct {
	client *etcd.Client
}

func NewEtcdRepo() (*etcdRepo, error) {
	client, err := stores.NewEtcdClient()
	if err != nil {
		return nil, err
	}

	return &etcdRepo{client: client}, nil
}

func (e *etcdRepo) Create(app *App) (*App, error) {
	b, err := json.Marshal(app)
	if err != nil {
		return nil, err
	}

	_, err = e.client.Set(e.key(app.Name), string(b), 0)
	return app, err
}

func (e *etcdRepo) FindByName(name Name) (*App, error) {
	r, err := e.client.Get(e.key(name), false, false)
	if err != nil {
		return nil, err
	}

	app := &App{}
	err = json.Unmarshal([]byte(r.Node.Value), app)
	return app, err
}

func (e *etcdRepo) FindByRepo(repo repos.Repo) (*App, error) {
	r, err := e.client.Get(e.key(Name("")), false, false)
	if err != nil {
		return nil, err
	}

	var a = &App{}

	for _, n := range r.Node.Nodes {
		err = json.Unmarshal([]byte(n.Value), a)
		if err != nil {
			return nil, err
		}
		if a.Repo == repo {
			return a, nil
		}
	}

	return nil, nil
}

func (e *etcdRepo) key(name Name) string {
	return fmt.Sprintf("/empire/apps/%s", name)
}
