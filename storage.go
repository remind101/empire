package empire

type Storage interface {
	AppsFind(AppsQuery) (*App, error)
	Apps(AppsQuery) ([]*App, error)
}
