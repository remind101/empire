package empire

import (
	"database/sql"
	"errors"

	"time"

	"github.com/remind101/pkg/timex"
	"gopkg.in/gorp.v1"
)

var (
	ErrDomainInUse        = errors.New("Domain currently in use by another app.")
	ErrDomainAlreadyAdded = errors.New("Domain already added to this app.")
	ErrDomainNotFound     = errors.New("Domain could not be found.")
)

type Domain struct {
	ID        string    `db:"id"`
	AppID     string    `db:"app_id"`
	Hostname  string    `db:"hostname"`
	CreatedAt time.Time `db:"created_at"`
}

// PreInsert implements a pre insert hook for the db interface
func (d *Domain) PreInsert(s gorp.SqlExecutor) error {
	d.CreatedAt = timex.Now()
	return nil
}

type domainRegistry interface {
	Register(*Domain) error
	Unregister(*Domain) error
}

func newDomainRegistry(urls string) domainRegistry {
	if urls == "fake" {
		return &fakeDomainRegistry{}
	}

	// TODO: Add a Route53 Registry that will create a CNAME to the app's ELB.
	return &fakeDomainRegistry{}
}

type fakeDomainRegistry struct{}

func (r *fakeDomainRegistry) Register(domain *Domain) error {
	return nil
}

func (r *fakeDomainRegistry) Unregister(domain *Domain) error {
	return nil
}

type domainsService struct {
	store    *store
	registry domainRegistry
}

func (s *domainsService) DomainsCreate(domain *Domain) (*Domain, error) {
	d, err := s.store.DomainsFindByHostname(domain.Hostname)
	if err != nil {
		return domain, err
	}

	if d != nil {
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

	if err := s.registry.Register(domain); err != nil {
		return domain, err
	}

	return domain, err
}

func (s *domainsService) DomainsDestroy(domain *Domain) error {
	if err := s.registry.Unregister(domain); err != nil {
		return err
	}

	if err := s.store.DomainsDestroy(domain); err != nil {
		return err
	}

	// If app has no domains associated, make it private
	d, err := s.store.DomainsFindByApp(&App{ID: domain.AppID})
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
	a, err := s.store.AppsFind(appID)
	if err != nil {
		return err
	}

	a.Exposure = "public"
	if _, err := s.store.AppsUpdate(a); err != nil {
		return err
	}

	return nil
}

func (s *domainsService) makePrivate(appID string) error {
	a, err := s.store.AppsFind(appID)
	if err != nil {
		return err
	}

	a.Exposure = "private"
	if _, err := s.store.AppsUpdate(a); err != nil {
		return err
	}

	return nil
}

func (s *store) DomainsFindByApp(app *App) ([]*Domain, error) {
	return domainsFindByApp(s.db, app)
}

func (s *store) DomainsFindByHostname(hostname string) (*Domain, error) {
	return domainsFindByHostname(s.db, hostname)
}

func (s *store) DomainsCreate(domain *Domain) (*Domain, error) {
	return domainsCreate(s.db, domain)
}

func (s *store) DomainsDestroy(domain *Domain) error {
	return domainsDestroy(s.db, domain)
}

func domainsFindByApp(db *db, app *App) ([]*Domain, error) {
	var domains []*Domain
	return domains, db.Select(&domains, `select * from domains where app_id = $1 order by hostname`, app.ID)
}

func domainsFindByHostname(db *db, hostname string) (*Domain, error) {
	var domain Domain
	if err := db.SelectOne(&domain, `select * from domains where hostname = $1`, hostname); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return &domain, nil
}

func domainsCreate(db *db, domain *Domain) (*Domain, error) {
	return domain, db.Insert(domain)
}

func domainsDestroy(db *db, domain *Domain) error {
	_, err := db.Delete(domain)
	return err
}
