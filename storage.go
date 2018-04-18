package empire

type Storage interface {
	AppsFind(AppsQuery) (*App, error)
	Apps(AppsQuery) ([]*App, error)
	AppsDestroy(*App) error

	ReleasesFind(ReleasesQuery) (*Release, error)
	Releases(ReleasesQuery) ([]*Release, error)
	ReleasesCreate(*App, string) (*Release, error)

	Reset() error
	IsHealthy() error
}
