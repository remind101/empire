package empire

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Commit represents a git commit on a repo. Commits can be deployed.
type Commit struct {
	Repo Repo
	Sha  string
}

// CommitResolver is an interface that can resolve a Commit to an Image.
type CommitResolver interface {
	Resolve(Commit) (Image, error)
}

type resolver struct{}

func (r *resolver) Resolve(Commit) (Image, error) {
	return Image{}, nil
}

// RegistryResolver is an implementation of the CommitResolver interface that
// resolves a Commit to an Image, by finding the docker image that's tagged with
// the git sha.
type RegistryResolver struct {
	Registry string
	client   *http.Client
}

func (r *RegistryResolver) Resolve(commit Commit) (Image, error) {
	image := Image{
		Repo: commit.Repo,
	}

	url := fmt.Sprintf(
		"http://%s/v1/repositories/%s/tags/%s",
		r.Registry, commit.Repo, commit.Sha,
	)

	if r.client == nil {
		r.client = http.DefaultClient
	}

	resp, err := r.client.Get(url)
	if err != nil {
		return image, err
	}

	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return image, err
	}
	resp.Body.Close()

	var id string
	if err := json.Unmarshal(raw, &id); err != nil {
		fmt.Println(err)
		return image, err
	}

	image.ID = id

	return image, nil
}

type GitHubDeploysService struct {
	*DeploysService
	resolver CommitResolver
}

func (s *GitHubDeploysService) DeployCommit(commit Commit) (*Deploy, error) {
	image, err := s.resolver.Resolve(commit)
	if err != nil {
		return nil, err
	}

	return s.DeployImage(image)
}
