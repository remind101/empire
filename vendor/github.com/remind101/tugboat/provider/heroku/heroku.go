package heroku

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"code.google.com/p/goauth2/oauth"

	"github.com/google/go-github/github"
	"github.com/remind101/tugboat"
	"github.com/remind101/tugboat/pkg/heroku"
	"golang.org/x/net/context"
)

// Provider is a tugboat.Provider that can perform deployments using the Heroku
// platform API.
type Provider struct {
	heroku *heroku.Service
	github *github.Client
}

func NewProvider(githubToken, herokuToken string) *Provider {
	heroku := newHerokuClient(herokuToken)
	github := newGitHubClient(githubToken)

	return &Provider{
		heroku: heroku,
		github: github,
	}
}

func (p *Provider) Name() string {
	return "heroku"
}

func (p *Provider) Deploy(ctx context.Context, d *tugboat.Deployment, w io.Writer) error {
	sha := d.Sha

	fmt.Fprintf(w, "-----> Fetching archive link for %s@%s... ", d.Repo, d.Sha)

	source, err := p.getSource(d.Repo, sha)
	if err != nil {
		return err
	}

	app := appFor(d)
	fmt.Fprintf(w, "done.\n-----> Deploying to %s...\n", app)

	b, err := p.buildCreate(app, source, sha)
	if err != nil {
		return fmt.Errorf("Unable to create build: %s", err)
	}

	resp, err := http.Get(b.OutputStreamURL)
	if err != nil {
		return fmt.Errorf("Unable to get log stream at %s: %s", b.OutputStreamURL, err)
	}
	defer resp.Body.Close()

	if _, err := io.Copy(w, resp.Body); err != nil {
		return err
	}

	br, err := p.buildResult(app, b.ID)
	if err != nil {
		return fmt.Errorf("Unable to get build result: %s", err)
	}

	if br.Build.Status == "failed" {
		return tugboat.ErrFailed
	}

	return nil
}

func (p *Provider) getSource(repo, ref string) (string, error) {
	sp := strings.Split(repo, "/")
	owner := sp[0]
	name := sp[1]

	u, resp, err := p.github.Repositories.GetArchiveLink(owner, name, github.Tarball, &github.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		return "", nil
	}

	if u == nil || resp.StatusCode != 302 {
		return "", errors.New("could not get archive link")
	}

	return u.String(), nil
}

func (p *Provider) buildCreate(app, source, sha string) (*heroku.Build, error) {
	return p.heroku.BuildCreate(app, heroku.BuildCreateOpts{
		SourceBlob: struct {
			URL     *string `json:"url,omitempty"`
			Version *string `json:"version,omitempty"`
		}{&source, &sha},
	})
}

func (p *Provider) buildResult(app string, buildID string) (*heroku.BuildResult, error) {
	return p.heroku.BuildResultInfo(app, buildID)
}

func appFor(d *tugboat.Deployment) string {
	sp := strings.Split(d.Repo, "/")
	repo := sp[1]
	env := d.Environment

	if env == "production" {
		return repo
	}

	return repo + "-" + env
}

func newHerokuClient(token string) *heroku.Service {
	t := &heroku.Transport{
		Version:  "edge",
		Password: token,
	}

	return heroku.NewService(&http.Client{
		Transport: t,
	})
}

func newGitHubClient(token string) *github.Client {
	t := &oauth.Transport{
		Token: &oauth.Token{AccessToken: token},
	}

	return github.NewClient(t.Client())
}
