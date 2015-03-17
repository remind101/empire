package empire

import (
	"fmt"
)

// DeployID represents the unique identifier for a Deploy.
type DeployID string

// Deploy represents a deployment to the platform.
type Deploy struct {
	ID      DeployID
	Status  string
	Release *Release
}

type Commit struct {
	Repo Repo
	Sha  string
}

type deployer struct {
	// Organization is a docker repo organization to fallback to if the app
	// doesn't specify a docker repo.
	Organization string

	*appsService
	*configsService
	*slugsService
	*releasesService
}

func (s *deployer) DeployImageToApp(app *App, image Image) (*Deploy, error) {
	if err := s.appsService.AppsEnsureRepo(app, DockerRepo, image.Repo); err != nil {
		return nil, err
	}

	// Grab the latest config.
	config, err := s.configsService.ConfigsCurrent(app)
	if err != nil {
		return nil, err
	}

	// Create a new slug for the docker image.
	//
	// TODO This is actually going to be pretty slow, so
	// we'll need to do
	// some polling or events/webhooks here.
	slug, err := s.slugsService.SlugsCreateByImage(image)
	if err != nil {
		return nil, err
	}

	// Create a new release for the Config
	// and Slug.
	desc := fmt.Sprintf("Deploy %s", image.String())
	release, err := s.releasesService.ReleasesCreate(app, config, slug, desc)
	if err != nil {
		return nil, err
	}

	// We're deployed! ...
	// hopefully.
	return &Deploy{
		ID:      "1",
		Release: release,
	}, nil
}

// Deploy deploys an Image to the cluster.
func (s *deployer) DeployImage(image Image) (*Deploy, error) {
	app, err := s.appsService.AppsFindOrCreateByRepo(DockerRepo, image.Repo)
	if err != nil {
		return nil, err
	}

	return s.DeployImageToApp(app, image)
}

// Deploy commit deploys the commit to a specific app.
func (s *deployer) DeployCommitToApp(app *App, commit Commit) (*Deploy, error) {
	var docker Repo

	if err := s.appsService.AppsEnsureRepo(app, GitHubRepo, commit.Repo); err != nil {
		return nil, err
	}

	if app.Repos.Docker != nil {
		docker = *app.Repos.Docker
	} else {
		docker = s.fallbackRepo(app.Name)
	}

	return s.DeployImageToApp(app, Image{
		Repo: docker,
		ID:   commit.Sha,
	})
}

// DeployCommit resolves the Commit to an Image then deploys the Image.
func (s *deployer) DeployCommit(commit Commit) (*Deploy, error) {
	app, err := s.appsService.AppsFindOrCreateByRepo(GitHubRepo, commit.Repo)
	if err != nil {
		return nil, err
	}

	return s.DeployCommitToApp(app, commit)
}

func (s *deployer) fallbackRepo(appName string) Repo {
	return Repo(fmt.Sprintf("%s/%s", s.Organization, appName))
}
