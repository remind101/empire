package empire

import (
	"regexp"
	"strings"

	"github.com/remind101/empire/pkg/image"
)

// NamePattern is a regex pattern that app names must conform to.
var NamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{2,30}$`)

// appNameFromRepo generates a name from a Repo
//
//	remind101/r101-api => r101-api
func appNameFromRepo(repo string) string {
	p := strings.Split(repo, "/")
	return p[len(p)-1]
}

// App represents an Empire application.
type App struct {
	// An auto incremented ID representing the version of this app.
	Version int

	// The name of the application.
	Name string

	// Maintenance defines whether the app is in maintenance mode or not.
	Maintenance bool

	// The environment variables for this application.
	Environment map[string]string

	// The "slug" for this application (Docker Image).
	Image *image.Image

	// The app repo's git commit hash related to this Docker image.
	GitSHA string

	// The process formation for this application.
	Formation Formation
}

func NewApp(name string) *App {
	return &App{
		Name:        name,
		Environment: make(map[string]string),
		Formation:   make(Formation),
		Image: &image.Image{
			Repository: "#none",
		},
	}
}

// IsValid returns an error if the app isn't valid.
func (a *App) IsValid() error {
	if !NamePattern.Match([]byte(a.Name)) {
		return ErrInvalidName
	}

	return nil
}

func (a *App) BeforeCreate() error {
	return a.IsValid()
}

// AppsQuery is a scope implementation for common things to filter releases
// by.
type AppsQuery struct {
	// If provided, finds apps matching the given name.
	Name *string
}
