package tugboat

import (
	"encoding/json"
	"strings"

	"github.com/google/go-github/github"
)

// statusUpdater is an interface that's used to notify external services about
// the status of a deployment.
type statusUpdater interface {
	// UpdateStatus should update an external service about the new status
	// of the Deployment.
	UpdateStatus(*Deployment) error
}

// pusherUpdater is a statusUpdater implementation that notifies pusher.
type pusherUpdater struct {
	pusher Pusher
}

func (u *pusherUpdater) UpdateStatus(d *Deployment) error {
	channel := deploymentChannel(d.ID)

	data := struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}{
		ID:     d.ID,
		Status: d.Status.String(),
	}

	raw, err := json.Marshal(&data)
	if err != nil {
		return err
	}

	return u.pusher.Publish(string(raw), "status", channel)
}

type githubClient interface {
	CreateDeploymentStatus(owner, repo string, deployment int, request *github.DeploymentStatusRequest) (*github.DeploymentStatus, *github.Response, error)
}

// maps a DeploymentStatus to a github deployment status string.
var githubStatus = map[DeploymentStatus]string{
	StatusStarted:   "pending",
	StatusSucceeded: "success",
	StatusErrored:   "error",
	StatusFailed:    "failure",
}

// githubUpdater is a statusUpdater implementation that creates deployment
// statuses on github for the commit.
type githubUpdater struct {
	github githubClient
}

func (u *githubUpdater) UpdateStatus(d *Deployment) error {
	sp := strings.Split(d.Repo, "/")
	owner := sp[0]
	repo := sp[1]

	_, _, err := u.github.CreateDeploymentStatus(owner, repo, int(d.GitHubID), &github.DeploymentStatusRequest{
		State:       github.String(githubStatus[d.Status]),
		TargetURL:   github.String(d.URL()),
		Description: github.String(d.Provider),
	})

	return err
}

type multiUpdater []statusUpdater

func (u multiUpdater) UpdateStatus(d *Deployment) error {
	for _, updater := range u {
		if err := updater.UpdateStatus(d); err != nil {
			return err
		}
	}

	return nil
}
