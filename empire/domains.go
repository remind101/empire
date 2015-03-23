package empire

import (
	"database/sql"
	"time"

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
