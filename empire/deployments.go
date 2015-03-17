package empire

import (
	"fmt"
)

// Deployment statuses.
const (
	StatusPending = "pending"
	StatusFailed  = "failed"
	StatusSuccess = "success"
)

// Deployment represents a deployment to the platform.
type Deployment struct {
	ID      string `db:"id"`
	AppName string `db:"app_id"`
	Status  string `db:"status"`
	Image   Image  `db:"image"`

	ReleaseID *string `db:"release_id"`

	// Used to store the old status when changing statuses.
	prevStatus string `db:"-"`
}

// Success marks the deployment as successful. The release provided will be
// associated with this deployment.
func (d *Deployment) Success(release *Release) *Deployment {
	d.ReleaseID = &release.ID
	d.prevStatus, d.Status = d.Status, StatusSuccess
	return d
}

type Commit struct {
	Repo Repo
	Sha  string
}

func (s *store) DeploymentsCreate(d *Deployment) (*Deployment, error) {
	return deploymentsCreate(s.db, d)
}

func (s *store) DeploymentsUpdate(d *Deployment) error {
	return deploymentsUpdate(s.db, d)
}

type deploymentsService struct {
	store *store
}

func (s *deploymentsService) DeploymentsCreate(d *Deployment) (*Deployment, error) {
	d.Status = StatusPending
	return s.store.DeploymentsCreate(d)
}

func (s *deploymentsService) DeploymentsUpdate(d *Deployment) error {
	return s.store.DeploymentsUpdate(d)
}

type deployer struct {
	// Organization is a docker repo organization to fallback to if the app
	// doesn't specify a docker repo.
	Organization string

	*deploymentsService
	*appsService
	*configsService
	*slugsService
	*releasesService
}

func (s *deployer) DeployImageToApp(app *App, image Image) (*Deployment, error) {
	if err := s.appsService.AppsEnsureRepo(app, DockerRepo, image.Repo); err != nil {
		return nil, err
	}

	d, err := s.DeploymentsCreate(&Deployment{
		AppName: app.Name,
		Image:   image,
	})
	if err != nil {
		return d, err
	}

	// Grab the latest config.
	config, err := s.ConfigsCurrent(app)
	if err != nil {
		return d, err
	}

	// Create a new slug for the docker image.
	//
	// TODO This is actually going to be pretty slow, so
	// we'll need to do
	// some polling or events/webhooks here.
	slug, err := s.SlugsCreateByImage(image)
	if err != nil {
		return d, err
	}

	// Create a new release for the Config
	// and Slug.
	desc := fmt.Sprintf("Deploy %s", image.String())
	release, err := s.ReleasesCreate(app, config, slug, desc)
	if err != nil {
		return d, err
	}

	// Mark the deployment as successful.
	if err := s.DeploymentsUpdate(d.Success(release)); err != nil {
		return d, err
	}

	return d, nil
}

// Deploy deploys an Image to the cluster.
func (s *deployer) DeployImage(image Image) (*Deployment, error) {
	app, err := s.appsService.AppsFindOrCreateByRepo(DockerRepo, image.Repo)
	if err != nil {
		return nil, err
	}

	return s.DeployImageToApp(app, image)
}

// Deploy commit deploys the commit to a specific app.
func (s *deployer) DeployCommitToApp(app *App, commit Commit) (*Deployment, error) {
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
func (s *deployer) DeployCommit(commit Commit) (*Deployment, error) {
	app, err := s.appsService.AppsFindOrCreateByRepo(GitHubRepo, commit.Repo)
	if err != nil {
		return nil, err
	}

	return s.DeployCommitToApp(app, commit)
}

func (s *deployer) fallbackRepo(appName string) Repo {
	return Repo(fmt.Sprintf("%s/%s", s.Organization, appName))
}

func deploymentsCreate(db *db, d *Deployment) (*Deployment, error) {
	return d, db.Insert(d)
}

func deploymentsUpdate(db *db, d *Deployment) error {
	_, err := db.Update(d)
	return err
}
