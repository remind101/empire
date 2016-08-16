package empire

import (
	"context"

	"github.com/jinzhu/gorm"
)

type certsService struct {
	*Empire
}

func (s *certsService) CertsAttach(ctx context.Context, db *gorm.DB, app *App, cert string) error {
	app.Cert = cert

	if err := appsUpdate(db, app); err != nil {
		return err
	}

	if err := s.releases.Restart(ctx, db, app); err != nil {
		if err == ErrNoReleases {
			return nil
		}

		return err
	}

	return nil
}
