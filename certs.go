package empire

import (
	"github.com/jinzhu/gorm"
	"golang.org/x/net/context"
)

type certsService struct {
	*Empire
}

func (s *certsService) CertsAttach(ctx context.Context, db *gorm.DB, opts CertsAttachOpts) error {
	app := opts.App
	if app.Certs == nil {
		app.Certs = make(Certs)
	}

	process := opts.Process
	if process == "" {
		process = webProcessType
	}

	app.Certs[process] = opts.Cert

	if err := appsUpdate(db, app); err != nil {
		return err
	}

	if err := s.releases.ReleaseApp(ctx, db, app, nil); err != nil {
		if err == ErrNoReleases {
			return nil
		}

		return err
	}

	return nil
}
