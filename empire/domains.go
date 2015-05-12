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
	d, err := s.store.DomainsFind(DomainHostname(domain.Hostname))
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

	return domain, err
}

func (s *domainsService) DomainsDestroy(domain *Domain) error {
	if err := s.store.DomainsDestroy(domain); err != nil {
		return err
	}

	// If app has no domains associated, make it private
	d, err := s.store.DomainsAll(DomainApp(domain.App))
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
	a, err := s.store.AppsFind(AppID(appID))
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
	a, err := s.store.AppsFind(AppID(appID))
	if err != nil {
		return err
	}

	a.Exposure = "private"
	if err := s.store.AppsUpdate(a); err != nil {
		return err
	}

	return nil
}

// DomainHostname returns a scope that finds a domain by hostname.
func DomainHostname(hostname string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("hostname = ?", hostname)
	}
}

// DomainApp returns a scope that will find domains for a given app.
func DomainApp(app *App) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("app_id = ?", app.ID)
	}
}

func (s *store) DomainsFind(scope func(*gorm.DB) *gorm.DB) (*Domain, error) {
	var domain Domain
	if err := s.db.Scopes(scope).First(&domain).Error; err != nil {
		if err == gorm.RecordNotFound {
			return nil, nil
		}

		return nil, err
	}
	return &domain, nil
}

func (s *store) DomainsAll(scope func(*gorm.DB) *gorm.DB) ([]*Domain, error) {
	var domains []*Domain
	return domains, s.db.Scopes(scope).Find(&domains).Error
}

func (s *store) DomainsCreate(domain *Domain) (*Domain, error) {
	return domainsCreate(s.db, domain)
}

func (s *store) DomainsDestroy(domain *Domain) error {
	return domainsDestroy(s.db, domain)
}

func domainsCreate(db *gorm.DB, domain *Domain) (*Domain, error) {
	return domain, db.Create(domain).Error
}

func domainsDestroy(db *gorm.DB, domain *Domain) error {
	return db.Delete(domain).Error
}
