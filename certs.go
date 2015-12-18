package empire

import "golang.org/x/net/context"

type certsService struct {
	*Empire
}

func (s *certsService) CertsAttach(ctx context.Context, app *App, cert string) error {
	app.Cert = cert

	if err := s.store.AppsUpdate(app); err != nil {
		return err
	}

	if err := s.releaser.ReleaseApp(ctx, app); err != nil {
		if err == ErrNoReleases {
			return nil
		}

		return err
	}

	return nil
}
