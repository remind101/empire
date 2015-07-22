package tugboattest

import (
	"net/http"
	"os"
	"testing"

	"github.com/ejholmes/flock"
	"github.com/remind101/tugboat"
	"github.com/remind101/tugboat/provider/fake"
	"github.com/remind101/tugboat/server"
)

const GitHubSecret = "abcd"

func New(t testing.TB) *tugboat.Tugboat {
	config := tugboat.Config{}
	config.DB = "postgres://localhost/tugboat?sslmode=disable"

	tug, err := tugboat.New(config)
	if err != nil {
		t.Fatal(err)
	}
	tug.Providers = []tugboat.Provider{fake.NewProvider()}

	if err := tug.Reset(); err != nil {
		t.Fatal(err)
	}

	return tug
}

func NewServer(tug *tugboat.Tugboat) http.Handler {
	config := server.Config{}
	config.GitHub.Secret = GitHubSecret
	return server.New(tug, config)
}

var dblock = "/tmp/tugboat.lock"

// Run runs testing.M after aquiring a lock against the database.
func Run(m *testing.M) {
	l, err := flock.NewPath(dblock)
	if err != nil {
		panic(err)
	}

	l.Lock()
	defer l.Unlock()

	os.Exit(m.Run())
}
