package empire

type mockreleasesService struct {
	releasesService // Just to satisfy the interface.

	ReleasesCreateFunc func(*App, *Config, *Slug, string) (*Release, error)
}

func (s *mockreleasesService) ReleasesCreate(app *App, config *Config, slug *Slug, desc string) (*Release, error) {
	if s.ReleasesCreateFunc != nil {
		return s.ReleasesCreateFunc(app, config, slug, desc)
	}

	return nil, nil
}
