package empire

import (
	"errors"

	"time"

	"github.com/jinzhu/gorm"
	"github.com/remind101/pkg/timex"
)

var (
	ErrDomainInUse        = errors.New("Domain currently in use by another app.")
	ErrDomainAlreadyAdded = errors.New("Domain already added to this app.")
	ErrDomainNotFound     = errors.New("Domain could not be found.")
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
	store *store
}

func (s *domainsService) DomainsCreate(domain *Domain) (*Domain, error) {
	d, err := s.store.DomainsFirst(DomainsQuery{Hostname: &domain.Hostname})
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

	_, err = s.store.DomainsCreate(domain)
	if err != nil {
		return domain, err
	}

	if err := s.makePublic(domain.AppID); err != nil {
		return domain, err
	}

	return domain, err
}

func (s *domainsService) DomainsDestroy(domain *Domain) error {
	if err := s.store.DomainsDestroy(domain); err != nil {
		return err
	}

	// If app has no domains associated, make it private
	d, err := s.store.Domains(DomainsQuery{App: domain.App})
	if err != nil {
		return err
	}

	if len(d) == 0 {
		if err := s.makePrivate(domain.AppID); err != nil {
			return err
		}
	}

	return nil
}

func (s *domainsService) makePublic(appID string) error {
	a, err := s.store.AppsFirst(AppsQuery{ID: &appID})
	if err != nil {
		return err
	}

	a.Exposure = "public"
	if err := s.store.AppsUpdate(a); err != nil {
		return err
	}

	return nil
}

func (s *domainsService) makePrivate(appID string) error {
	a, err := s.store.AppsFirst(AppsQuery{ID: &appID})
	if err != nil {
		return err
	}

	a.Exposure = "private"
	if err := s.store.AppsUpdate(a); err != nil {
		return err
	}

	return nil
}

// DomainsQuery is a Scope implementation for common things to filter releases
// by.
type DomainsQuery struct {
	// If provided, finds domains matching the given hostname.
	Hostname *string

	// If provided, filters domains belonging to the given app.
	App *App
}

// Scope implements the Scope interface.
func (q DomainsQuery) Scope(db *gorm.DB) *gorm.DB {
	var scope ComposedScope

	if q.Hostname != nil {
		scope = append(scope, FieldEquals("hostname", *q.Hostname))
	}

	if q.App != nil {
		scope = append(scope, FieldEquals("app_id", q.App.ID))
	}

	return scope.Scope(db)
}

// DomainsFirst returns the first matching domain.
func (s *store) DomainsFirst(scope Scope) (*Domain, error) {
	var domain Domain
	return &domain, s.First(scope, &domain)
}

// Domains returns all domains matching the scope.
func (s *store) Domains(scope Scope) ([]*Domain, error) {
	var domains []*Domain
	return domains, s.Find(scope, &domains)
}

// DomainsCreate persists the Domain.
func (s *store) DomainsCreate(domain *Domain) (*Domain, error) {
	return domainsCreate(s.db, domain)
}

// DomainsDestroy destroys the Domain.
func (s *store) DomainsDestroy(domain *Domain) error {
	return domainsDestroy(s.db, domain)
}

func domainsCreate(db *gorm.DB, domain *Domain) (*Domain, error) {
	return domain, db.Create(domain).Error
}

func domainsDestroy(db *gorm.DB, domain *Domain) error {
	return db.Delete(domain).Error
}
