package heroku

import "github.com/remind101/empire/empire"

type Empire interface {
	SlugsFind(id string) (*empire.Slug, error)
	ProcessesAll(*empire.Release) (empire.Formation, error)

	JobStatesByApp(*empire.App) ([]*empire.JobState, error)

	// TODO Remove these
	ReleasesCreate(*empire.App, *empire.Config, *empire.Slug, string) (*empire.Release, error)
	ReleasesLast(*empire.App) (*empire.Release, error)
	ReleasesFindByAppAndVersion(*empire.App, int) (*empire.Release, error)
	ReleasesFindByApp(*empire.App) ([]*empire.Release, error)

	empire.AppsService
	empire.ConfigsService
	empire.DeploysService
	empire.Manager
	empire.AccessTokensService
}
