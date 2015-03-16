package heroku

import "github.com/remind101/empire/empire"

type Empire interface {
	SlugsFind(id string) (*empire.Slug, error)
	ProcessesAll(*empire.Release) (empire.Formation, error)

	JobStatesByApp(*empire.App) ([]*empire.JobState, error)

	AppsAll() ([]*empire.App, error)
	AppsDestroy(*empire.App) error
	AppsCreate(*empire.App) (*empire.App, error)
	AppsFind(name string) (*empire.App, error)

	// TODO Remove these
	ReleasesCreate(*empire.App, *empire.Config, *empire.Slug, string) (*empire.Release, error)
	ReleasesLast(*empire.App) (*empire.Release, error)
	ReleasesFindByAppAndVersion(*empire.App, int) (*empire.Release, error)
	ReleasesFindByApp(*empire.App) ([]*empire.Release, error)

	empire.ConfigsService
	empire.DeploysService
	empire.Manager
	empire.AccessTokensService
}
