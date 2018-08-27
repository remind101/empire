package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"gopkg.in/src-d/go-billy.v4/memfs"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/storage/memory"

	"github.com/remind101/empire"
)

const (
	FileVersion  = "VERSION"
	FileEnv      = "app.env"
	FileImage    = "image.txt"
	FileServices = "services.json"
)

// Storage is an implementation of the empire.Storage interface that uses the
// GitHub Git API to store application configuration withing a GitHub
// repository.
//
// https://developer.github.com/v3/git/
// https://developer.github.com/v3/repos/
type Storage struct {
	// When provided, this will be used as the committer of all commits
	// made. If not provided, commits will be attributed to the
	// authenticated App/User.
	Committer *object.Signature

	// The git URL to clone/push from/to.
	URL string

	// The base file path for where files will be committed.
	BasePath string

	// Ref to update after creating a commit.
	Ref string

	auth transport.AuthMethod
}

// NewStorage returns a new Storage instance usign a github client that's
// authenticated with the given http.Client
func NewStorage(auth transport.AuthMethod) *Storage {
	return &Storage{
		auth: auth,
	}
}

// ReleasesCreate creates a new release by making a commit to the GitHub
// repository. In CLI terminology, it's roughly equivalent to the following:
//
//	> git checkout -b changes
//	> touch app.json app.env image.txt services.json
//	> git commit -m "Description of the changes"
//	> git checkout base-ref
//	> git merge --no-ff changes
func (s *Storage) ReleasesCreate(ctx context.Context, w io.Writer, app *empire.App, event empire.Event) (*empire.Release, error) {
	// Auto increment the version number for this new release.
	app.Version = app.Version + 1

	fs := memfs.New()
	storer := memory.NewStorage()

	repo, err := git.CloneContext(ctx, storer, fs, &git.CloneOptions{
		URL:           s.URL,
		Auth:          s.auth,
		ReferenceName: plumbing.ReferenceName(s.Ref),
		Progress:      w,
	})
	if err != nil {
		return nil, err
	}

	workTree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	fname := filepath.Join(s.BasePath, app.Name, FileVersion)
	f, err := fs.Create(fname)
	if err != nil {
		return nil, err
	}
	if _, err := f.Write([]byte(fmt.Sprintf("v%d", app.Version))); err != nil {
		return nil, err
	}

	if _, err := workTree.Add(fname); err != nil {
		return nil, err
	}

	commitMessage := fmt.Sprintf("%s\n\n%s", event.String(), event.Message())

	author := commitAuthor(event.User())
	committer := s.Committer
	if committer == nil {
		committer = author
	}

	commit, err := workTree.Commit(commitMessage, &git.CommitOptions{
		Author:    addNow(author),
		Committer: addNow(committer),
	})
	if err != nil {
		return nil, err
	}

	if _, err := repo.CommitObject(commit); err != nil {
		return nil, err
	}

	if err := repo.PushContext(ctx, &git.PushOptions{
		Auth:     s.auth,
		Progress: w,
	}); err != nil {
		return nil, err
	}

	return nil, nil
}

// Releases returns a list of the most recent releases for the give application.
// It does so by looking what commits to the app.json file in the app directory.
func (s *Storage) Releases(q empire.ReleasesQuery) ([]*empire.Release, error) {
	return nil, nil
}

// Apps returns a list of all apps matching q.
func (s *Storage) Apps(q empire.AppsQuery) ([]*empire.App, error) {
	return nil, nil
}

func filterApps(apps []*empire.App, q empire.AppsQuery) []*empire.App {
	if q.Name != nil {
		apps = filter(apps, func(app *empire.App) bool {
			return app.Name == *q.Name
		})
	}
	return apps
}

func filter(apps []*empire.App, fn func(*empire.App) bool) []*empire.App {
	var filtered []*empire.App
	for _, app := range apps {
		if fn(app) {
			filtered = append(filtered, app)
		}
	}
	return filtered
}

// AppsDestroy destroys the given app.
func (s *Storage) AppsDestroy(app *empire.App) error {
	return errors.New("AppsDestroy not implemented")
}

// AppsFind finds a single app that matches q, and loads it's configuration.
func (s *Storage) AppsFind(q empire.AppsQuery) (*empire.App, error) {
	return nil, nil
}

// ReleasesFind finds a release that matches q.
func (s *Storage) ReleasesFind(q empire.ReleasesQuery) (*empire.Release, error) {
	return nil, errors.New("ReleasesFind not implemented")
}

// Reset does nothing for the GitHub storage backend.
func (s *Storage) Reset() error {
	return errors.New("refusing to reset GitHub storage backend")
}

// IsHealthy always returns healthy for the GitHub storage backend.
func (s *Storage) IsHealthy() error {
	return nil
}

// commitAuthor returns a suitable object.Signature for an authenticated user.
func commitAuthor(user *empire.User) *object.Signature {
	name := user.FullName
	if name == "" {
		name = user.Name
	}

	email := user.Email
	if email == "" {
		email = name
	}

	return &object.Signature{
		Name:  name,
		Email: email,
	}
}

func addNow(s *object.Signature) *object.Signature {
	if s == nil {
		return nil
	}
	s.When = time.Now()
	return s
}
