package empire

type Storage interface {
	AppsFind(AppsQuery) (*App, error)
	Apps(AppsQuery) ([]*App, error)
	AppsCreate(*App) (*App, error)
	AppsDestroy(*App) error

	Begin() Storage
	Rollback() error
	Commit() error
}
