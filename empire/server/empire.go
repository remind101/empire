package server

import "github.com/remind101/empire/empire"

type Empire interface {
	empire.AppsService
	empire.ConfigsService
	empire.SlugsService
	empire.ReleasesService
	empire.ProcessesService
	empire.JobsService
	empire.JobStatesService
	empire.Manager

	DeployImage(empire.Image) (*empire.Deploy, error)
	DeployCommit(empire.Commit) (*empire.Deploy, error)
}
