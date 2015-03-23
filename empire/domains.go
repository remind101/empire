package empire

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"gopkg.in/gorp.v1"
)

type Domain struct {
	ID        string    `db:"id"`
	AppName   string    `db:"app_id"`
	Hostname  string    `db:"hostname"`
	CreatedAt time.Time `db:"created_at"`
}

// PreInsert implements a pre insert hook for the db interface
func (d *Domain) PreInsert(s gorp.SqlExecutor) error {
	d.CreatedAt = Now()
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

	return &etcdDomainRegistry{client: etcd.NewClient(strings.Split(urls, ","))}
}

type fakeDomainRegistry struct{}

func (r *fakeDomainRegistry) Register(domain *Domain) error {
	return nil
}

func (r *fakeDomainRegistry) Unregister(domain *Domain) error {
	return nil
}

type etcdDomainRegistry struct {
	client *etcd.Client
}

func (r *etcdDomainRegistry) Register(domain *Domain) error {
	_, err := r.client.Set(r.key(domain.AppName, domain.Hostname), domain.Hostname, 0)
	return err
}

func (r *etcdDomainRegistry) Unregister(domain *Domain) error {
	_, err := r.client.Delete(r.key(domain.AppName, domain.Hostname), false)
	return err
}

func (r *etcdDomainRegistry) key(app, host string) string {
	return fmt.Sprintf("/services/__domains__/%s/%s", app, host)
}

type domainsService struct {
	store    *store
	registry domainRegistry
}

func (s *domainsService) DomainsCreate(domain *Domain) (*Domain, error) {
	if err := s.registry.Register(domain); err != nil {
		return domain, err
	}

	_, err := s.store.DomainsCreate(domain)
	return domain, err
}

func (s *domainsService) DomainsDestroy(domain *Domain) error {
	if err := s.registry.Unregister(domain); err != nil {
		return err
	}

	return s.store.DomainsDestroy(domain)
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
	return domains, db.Select(&domains, `select * from domains where app_id = $1 order by hostname`, app.Name)
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
