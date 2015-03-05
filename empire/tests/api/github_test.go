package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/remind101/empire/empire"
)

type GitHubDeploymentForm struct {
	Repo string `json:"name"`
	Sha  string `json:"sha"`
}

func TestGitHubDeploy(t *testing.T) {
	_, s := NewTestClient(t)
	defer s.Close()

	d := mustGitHubDeploy(t, s.URL, DefaultCommit)

	if got, want := d.Release.Version, 1; got != want {
		t.Fatalf("Version => %v; want %v", got, want)
	}

	if got, want := d.Release.Description, "Deploy "+DefaultImage.String(); got != want {
		t.Fatalf("Description => %v; want %v", got, want)
	}
}

func mustGitHubDeploy(t testing.TB, url string, commit empire.Commit) Deploy {
	var (
		d Deploy
	)

	body := fmt.Sprintf(`{"name":"%s","sha":"%s"}`, commit.Repo, commit.Sha)
	req, err := http.NewRequest("POST", url+"/github", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("X-GitHub-Event", "deployment")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		t.Fatal(err)
	}

	return d
}
