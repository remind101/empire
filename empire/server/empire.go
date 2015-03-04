package server

import "github.com/remind101/empire/empire"

type Empire interface {
	empire.AppsService
	empire.ConfigsService
	empire.SlugsService
	empire.ReleasesService
	empire.ProcessesService
	empire.DeploysService
	empire.JobsService
	empire.JobStatesService
	empire.Manager
}
