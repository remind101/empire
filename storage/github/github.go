package github

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/dotenv"
	"github.com/remind101/empire/pkg/image"
)

// When interacting with the GitHub API, we expect "/" to be the directory
// separator.
const DirectorySeparator = "/"

// For blobs, the file mode should always be this value.
//
// https://developer.github.com/v3/git/trees/#create-a-tree
const BlobPerms = "100644"

const (
	FileVersion  = "VERSION"
	FileImage    = "IMAGE"
	FileHash     = "HASH"
	FileEnv      = "app.env"
	FileServices = "services.json"
)

// Committer returns a *github.CommitAuthor that gets attributed to the given
// email.
func Committer(email string) *github.CommitAuthor {
	return &github.CommitAuthor{
		Name:  github.String("GitHub App"),
		Email: github.String(email),
	}
}

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
	Committer *github.CommitAuthor

	// The GitHub repository where configuration will be stored.
	Owner, Repo string

	// The base file path for where files will be committed.
	BasePath string

	// Ref to update after creating a commit.
	Ref string

	github *github.Client
}

// NewStorage returns a new Storage instance usign a github client that's
// authenticated with the given http.Client
func NewStorage(c *http.Client) *Storage {
	return &Storage{
		github: github.NewClient(c),
	}
}

// ReleasesCreate creates a new release by making a commit to the GitHub
// repository. In CLI terminology, it's roughly equivalent to the following:
//
//	> git checkout -b changes
//	> touch app.env IMAGE VERSION services.json
//	> git commit -m "Description of the changes"
//	> git checkout base-ref
//	> git merge --no-ff changes
func (s *Storage) ReleasesCreate(app *empire.App, event empire.Event) (*empire.Release, error) {

	// Auto increment the version number for this new release.
	app.Version = app.Version + 1

	// Set the App hash from event.String().
	app.Hash = strings.Split(event.String(), ":")[1]

	// Get details about the ref we want to update.
	ref, _, err := s.github.Git.GetRef(s.Owner, s.Repo, s.Ref)
	if err != nil {
		return nil, fmt.Errorf("get %q ref: %v", s.Ref, err)
	}

	// Get the last commit on the ref we want to update. This will be used
	// as the base for our changes.
	lastCommit, _, err := s.github.Git.GetCommit(s.Owner, s.Repo, *ref.Object.SHA)
	if err != nil {
		return nil, fmt.Errorf("get last commit for %q: %v", *ref.Object.SHA, err)
	}

	// Generate our new tree object with our app configuration.
	treeEntries, err := s.treeEntries(app)
	if err != nil {
		return nil, fmt.Errorf("generating tree: %v", err)
	}

	// Create a new tree object, based on the last commit.
	tree, _, err := s.github.Git.CreateTree(s.Owner, s.Repo, *lastCommit.Tree.SHA, treeEntries)
	if err != nil {
		return nil, fmt.Errorf("creating tree: %v", err)
	}

	commitMessage := fmt.Sprintf("%s\n\n%s", event.String(), event.Message())

	author := commitAuthor(event.User())
	committer := s.Committer
	if committer == nil {
		committer = author
		author = nil
	}

	// Create a new commit object with our new tree.
	commit, _, err := s.github.Git.CreateCommit(s.Owner, s.Repo, &github.Commit{
		Message:   github.String(commitMessage),
		Tree:      tree,
		Parents:   []github.Commit{*lastCommit},
		Author:    author,
		Committer: committer,
	})
	if err != nil {
		return nil, fmt.Errorf("creating commit: %v", err)
	}

	// Finally, we merge our commit with the new tree into the existing tree
	// in our target ref. This will create a merge commit.
	_, _, err = s.github.Repositories.Merge(s.Owner, s.Repo, &github.RepositoryMergeRequest{
		Base: github.String(s.Ref),
		Head: commit.SHA,
	})
	if err != nil {
		return nil, fmt.Errorf("merging %q into %q: %v", *commit.SHA, s.Ref, err)
	}
	created_at := time.Now()
	return &empire.Release{
		App:         app,
		Description: commitMessage,
		CreatedAt:   &created_at,
	}, nil
}

// Releases returns a list of the most recent releases for the give application.
// It does so by looking what commits to the VERSION file in the app directory.
func (s *Storage) Releases(q empire.ReleasesQuery) ([]*empire.Release, error) {
	// grab app from RelaseQuery.
	app := q.App

	// Get a list of all commits that changed the VERSION file in the app directory.
	commits, _, err := s.github.Repositories.ListCommits(s.Owner, s.Repo, &github.CommitsListOptions{
		SHA:  s.Ref,
		Path: s.Path(app.Name, FileVersion),
	})
	if err != nil {
		return nil, err
	}

	var releases []*empire.Release

	// TODO(ejholmes): This loop is pretty inneficient right now since it's
	// N+1 and results in a lot of API calls to GitHub.
	for _, commit := range commits {
		f := s.GetContentsAtRef(*commit.SHA)

		// Only load the VERSION file when listing releases.
		include := map[string]bool{
			FileVersion: true,
			FileImage:   true,
		}

		app, err := loadApp(f, &empire.App{Name: app.Name}, include)
		if err != nil {
			return nil, err
		}

		// Use first line of the commit message as the release description.
		r := bufio.NewReader(strings.NewReader(*commit.Commit.Message))
		desc, _ := r.ReadString('\n')

		releases = append(releases, &empire.Release{
			App:         app,
			Description: strings.TrimSpace(desc),
			CreatedAt:   commit.Commit.Committer.Date,
			UserId:      *commit.Commit.Author.Name,
			UserEmail:   *commit.Commit.Author.Email,
		})
	}

	return releases, nil
}

// Apps returns a list of all apps matching q.
func (s *Storage) Apps(q empire.AppsQuery) ([]*empire.App, error) {
	_, directoryContent, _, err := s.GetContents()
	if err != nil {
		return nil, fmt.Errorf("get contents of %q in %q: %v", s.BasePath, s.Ref, err)
	}

	var apps []*empire.App
	for _, f := range directoryContent {
		if *f.Type == "dir" {
			apps = append(apps, &empire.App{Name: *f.Name})
		}
	}

	return filterApps(apps, q), nil
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
	apps, err := s.Apps(q)
	if err != nil {
		return nil, err
	}
	if len(apps) == 0 {
		return nil, &empire.NotFoundError{Err: errors.New("app not found")}
	}

	app := apps[0]

	return loadApp(s, app, nil)
}

// GetContents gets some dir/file content in the repo, under the BasePath.
func (s *Storage) GetContents(elem ...string) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error) {
	return s.GetContentsAtRef(s.Ref)(elem...)
}

// GetContents gets some dir/file content in the repo, under the BasePath.
func (s *Storage) GetContentsAtRef(ref string) contentFetcherFunc {
	return contentFetcherFunc(func(elem ...string) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error) {
		fullPath := s.Path(elem...)
		return s.github.Repositories.GetContents(s.Owner, s.Repo, fullPath, &github.RepositoryContentGetOptions{
			Ref: ref,
		})
	})
}

// ReleasesFind finds a release that matches q.
func (s *Storage) ReleasesFind(q empire.ReleasesQuery) (*empire.Release, error) {
	// pass the query struct and map of files to parse and include
	// to the Storage.Releases function.
	var releases, err = s.Releases(q)
	if err != nil {
		return nil, err
	}
	// loop over all Releases and return the release with the matching version.
	for _, release := range releases {
		if release.App.Version == *q.Version {
			return release, nil
		}
	}
	return nil, &empire.NotFoundError{Err: errors.New("release not found")}
}

// Reset does nothing for the GitHub storage backend.
func (s *Storage) Reset() error {
	return errors.New("refusing to reset GitHub storage backend")
}

// IsHealthy always returns healthy for the GitHub storage backend.
func (s *Storage) IsHealthy() error {
	return nil
}

func (s *Storage) Path(elem ...string) string {
	return PathJoin(s.BasePath, elem...)
}

// PathJoin joins the elem to basepath, in a way that disallows any path
// traversals in the GitHub API. This method:
//
// 1. Ensures that the returned path is _always_ under basepath.
// 2. Ensures that any directory separates in individual path components in elem
//    are stripped.
//
// Replacing this method with something like `PathJoin` would result in
// directory traversals.
func PathJoin(basepath string, elem ...string) string {
	var cleaned []string
	for _, e := range elem {
		cleaned = append(cleaned, url.QueryEscape(e))
	}
	return strings.Join(append([]string{basepath}, cleaned...), DirectorySeparator)
}

// treeEntries generates a list of github.TreeEntry describe the Empire App.
func (s *Storage) treeEntries(app *empire.App) ([]github.TreeEntry, error) {
	entries := []github.TreeEntry{
		{
			Path:    github.String(s.Path(app.Name, FileVersion)),
			Type:    github.String("blob"),
			Mode:    github.String(BlobPerms),
			Content: github.String(fmt.Sprintf("v%d", app.Version)),
		},
	}

	// The environment variables for this application.
	if app.Environment != nil {
		envFile := new(bytes.Buffer)
		if err := dotenv.Write(envFile, app.Environment); err != nil {
			return nil, err
		}
		entries = append(entries, github.TreeEntry{
			Path:    github.String(s.Path(app.Name, FileEnv)),
			Type:    github.String("blob"),
			Mode:    github.String(BlobPerms),
			Content: github.String(envFile.String()),
		})
	}

	if app.Image != nil {
		// The "slug" for this application (Docker Image).
		entries = append(entries, github.TreeEntry{
			Path:    github.String(s.Path(app.Name, FileImage)),
			Type:    github.String("blob"),
			Mode:    github.String(BlobPerms),
			Content: github.String(app.Image.String()),
		})
		// The app repo's git commit hash related to this Docker image.
		entries = append(entries, github.TreeEntry{
			Path:    github.String(s.Path(app.Name, FileHash)),
			Type:    github.String("blob"),
			Mode:    github.String(BlobPerms),
			Content: github.String(app.Hash),
		})
	}

	// The process formation for this application.
	if app.Formation != nil {
		formation, err := jsonMarshal(app.Formation)
		if err != nil {
			return nil, err
		}
		entries = append(entries, github.TreeEntry{
			Path:    github.String(s.Path(app.Name, FileServices)),
			Type:    github.String("blob"),
			Mode:    github.String(BlobPerms),
			Content: github.String(string(formation)),
		})
	}

	return entries, nil
}

func jsonMarshal(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

type contentFetcher interface {
	GetContents(...string) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error)
}

type contentFetcherFunc func(...string) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error)

func (fn contentFetcherFunc) GetContents(elem ...string) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error) {
	return fn(elem...)
}

var includeAll = map[string]bool{
	FileVersion:  true,
	FileImage:    true,
	FileEnv:      true,
	FileServices: true,
}

func loadApp(f contentFetcher, app *empire.App, include map[string]bool) (*empire.App, error) {
	if include == nil {
		include = includeAll
	}

	if include[FileVersion] {
		version, err := fileContent(f, PathJoin(app.Name, FileVersion))
		if err != nil {
			return nil, err
		}
		vi, err := strconv.Atoi(strings.TrimSpace(string(version))[1:])
		if err != nil {
			return nil, err
		}
		app.Version = vi
	}

	if include[FileServices] {
		if err := decodeFile(f, PathJoin(app.Name, FileServices), &app.Formation); err != nil {
			return nil, err
		}
	}

	if include[FileImage] {
		imageContent, err := fileContent(f, PathJoin(app.Name, FileImage))
		if err != nil {
			return nil, err
		}
		img, err := image.Decode(string(imageContent))
		if err != nil {
			return nil, err
		}
		app.Image = &img
	}

	if include[FileEnv] {
		envContent, err := fileContent(f, PathJoin(app.Name, FileEnv))
		if err != nil {
			return nil, err
		}
		env, err := dotenv.Read(bytes.NewReader(envContent))
		if err != nil {
			return nil, err
		}
		app.Environment = env
	}

	return app, nil
}

func decodeFile(f contentFetcher, path string, v interface{}) error {
	raw, err := fileContent(f, path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, &v)
}

func fileContent(f contentFetcher, path string) ([]byte, error) {
	fileContent, _, _, err := f.GetContents(path)
	if err != nil {
		return nil, fmt.Errorf("get contents of %q: %v", path, err)
	}

	raw, err := fileContent.Decode()
	if err != nil {
		return nil, fmt.Errorf("decoding %q: %v", path, err)
	}

	return raw, nil
}

// commitAuthor returns a suitable github.CommitAuthor for an authenticated
// user.
func commitAuthor(user *empire.User) *github.CommitAuthor {
	name := user.FullName
	if name == "" {
		name = user.Name
	}

	email := user.Email
	if email == "" {
		email = name
	}

	return &github.CommitAuthor{
		Name:  github.String(name),
		Email: github.String(email),
	}
}
