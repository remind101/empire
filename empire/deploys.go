package empire

import (
	"fmt"
)

// DeployID represents the unique identifier for a Deploy.
type DeployID string

// Deploy represents a deployment to the platform.
type Deploy struct {
	ID      DeployID `json:"id"`
	Status  string   `json:"status"`
	Release *Release `json:"release"`
}

type Commit struct {
	Repo Repo
	Sha  string
}

type ImageDeployer interface {
	DeployImage(Image) (*Deploy, error)
	DeployImageToApp(*App, Image) (*Deploy, error)
}

type CommitDeployer interface {
	DeployCommit(Commit) (*Deploy, error)
	DeployCommitToApp(*App, Commit) (*Deploy, error)
}

// DeploysService is an interface that can be implemented to deploy images.
type DeploysService interface {
	ImageDeployer
	CommitDeployer
}

// imageDeployer is a base implementation of the DeploysService
type imageDeployer struct {
	AppsService
	ConfigsService
	SlugsService
	ReleasesService
}

func (s *imageDeployer) DeployImageToApp(app *App, image Image) (*Deploy, error) {
	if err := s.AppsService.AppsEnsureRepo(app, DockerRepo, image.Repo); err != nil {
		return nil, err
	}

	// Grab the latest config.
	config, err := s.ConfigsService.ConfigsCurrent(app)
	if err != nil {
		return nil, err
	}

	// Create a new slug for the docker image.
	//
	// TODO This is actually going to be pretty slow, so
	// we'll need to do
	// some polling or events/webhooks here.
	slug, err := s.SlugsService.SlugsCreateByImage(image)
	if err != nil {
		return nil, err
	}

	// Create a new release for the Config
	// and Slug.
	desc := fmt.Sprintf("Deploy %s", image.String())
	release, err := s.ReleasesService.ReleasesCreate(app, config, slug, desc)
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
func (s *imageDeployer) DeployImage(image Image) (*Deploy, error) {
	app, err := s.AppsService.AppsFindOrCreateByRepo(DockerRepo, image.Repo)
	if err != nil {
		return nil, err
	}

	return s.DeployImageToApp(app, image)
}

// commitDeployer is an implementation of the CommitDeployer interface that uses
// a TagResolver to resolve the Commit to an Image before deploying.
type commitDeployer struct {
	ImageDeployer

	// Organization is a docker repo organization to fallback to if the app
	// doesn't specify a docker repo.
	Organization string

	appsService AppsService
}

// Deploy commit deploys the commit to a specific app.
func (s *commitDeployer) DeployCommitToApp(app *App, commit Commit) (*Deploy, error) {
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
		Tag:  commit.Sha,
	})
}

// DeployCommit resolves the Commit to an Image then deploys the Image.
func (s *commitDeployer) DeployCommit(commit Commit) (*Deploy, error) {
	app, err := s.appsService.AppsFindOrCreateByRepo(GitHubRepo, commit.Repo)
	if err != nil {
		return nil, err
	}

	return s.DeployCommitToApp(app, commit)
}

func (s *commitDeployer) fallbackRepo(appName string) Repo {
	return Repo(fmt.Sprintf("%s/%s", s.Organization, appName))
}
