package heroku

import "github.com/remind101/empire/empire"

type Empire interface {
	SlugsFind(id string) (*empire.Slug, error)

	empire.AppsService
	empire.ConfigsService
	empire.ReleasesService
	empire.ProcessesService
	empire.DeploysService
	empire.JobsService
	empire.JobStatesService
	empire.Manager
	empire.AccessTokensService
}
