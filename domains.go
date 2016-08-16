package empire

import (
	"context"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/remind101/pkg/timex"
)

type Domain struct {
	ID        string
	Hostname  string
	CreatedAt *time.Time

	AppID string
	App   *App
}

func (d *Domain) BeforeCreate() error {
	t := timex.Now()
	d.CreatedAt = &t
	return nil
}

type domainsService struct {
	*Empire
}

func (s *domainsService) DomainsCreate(ctx context.Context, db *gorm.DB, domain *Domain) (*Domain, error) {
	d, err := domainsFind(db, DomainsQuery{Hostname: &domain.Hostname})
	if err != nil && err != gorm.RecordNotFound {
		return domain, err
	}

	if err != gorm.RecordNotFound {
		if d.AppID == domain.AppID {
			return domain, ErrDomainAlreadyAdded
		} else {
			return domain, ErrDomainInUse
		}
	}

	_, err = domainsCreate(db, domain)
	if err != nil {
		return domain, err
	}

	if err := makePublic(db, domain.AppID); err != nil {
		return domain, err
	}

	return domain, err
}

func (s *domainsService) DomainsDestroy(ctx context.Context, db *gorm.DB, domain *Domain) error {
	if err := domainsDestroy(db, domain); err != nil {
		return err
	}

	// If app has no domains associated, make it private
	d, err := domains(db, DomainsQuery{App: domain.App})
	if err != nil {
		return err
	}

	if len(d) == 0 {
		if err := makePrivate(db, domain.AppID); err != nil {
			return err
		}
	}

	return nil
}

// DomainsQuery is a scope implementation for common things to filter releases
// by.
type DomainsQuery struct {
	// If provided, finds domains matching the given hostname.
	Hostname *string

	// If provided, filters domains belonging to the given app.
	App *App
}

// scope implements the scope interface.
func (q DomainsQuery) scope(db *gorm.DB) *gorm.DB {
	var scope composedScope

	if q.Hostname != nil {
		scope = append(scope, fieldEquals("hostname", *q.Hostname))
	}

	if q.App != nil {
		scope = append(scope, forApp(q.App))
	}

	return scope.scope(db)
}

// domainsFind returns the first matching domain.
func domainsFind(db *gorm.DB, scope scope) (*Domain, error) {
	var domain Domain
	return &domain, first(db, scope, &domain)
}

// domains returns all domains matching the scope.
func domains(db *gorm.DB, scope scope) ([]*Domain, error) {
	var domains []*Domain
	return domains, find(db, scope, &domains)
}

func domainsCreate(db *gorm.DB, domain *Domain) (*Domain, error) {
	return domain, db.Create(domain).Error
}

func domainsDestroy(db *gorm.DB, domain *Domain) error {
	return db.Delete(domain).Error
}

func makePublic(db *gorm.DB, appID string) error {
	a, err := appsFind(db, AppsQuery{ID: &appID})
	if err != nil {
		return err
	}

	a.Exposure = "public"
	if err := appsUpdate(db, a); err != nil {
		return err
	}

	return nil
}

func makePrivate(db *gorm.DB, appID string) error {
	a, err := appsFind(db, AppsQuery{ID: &appID})
	if err != nil {
		return err
	}

	a.Exposure = "private"
	if err := appsUpdate(db, a); err != nil {
		return err
	}

	return nil
}
