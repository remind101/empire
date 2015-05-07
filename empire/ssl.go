package empire

import (
	"database/sql"
	"time"

	"github.com/remind101/empire/empire/pkg/sslcert"
	"github.com/remind101/pkg/timex"
	"golang.org/x/net/context"
	"gopkg.in/gorp.v1"
)

type Certificate struct {
	ID               string    `db:"id"`
	AppID            string    `db:"app_id"`
	Name             string    `db:"name"`
	CertificateChain string    `db:"certificate_chain"`
	PrivateKey       string    `db:"-"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

// PreInsert implements a pre insert hook for the db interface
func (c *Certificate) PreInsert(s gorp.SqlExecutor) error {
	c.CreatedAt = timex.Now()
	c.UpdatedAt = c.CreatedAt
	return nil
}

// PreUpdate implements a pre insert hook for the db interface
func (c *Certificate) PreUpdate(s gorp.SqlExecutor) error {
	c.UpdatedAt = timex.Now()
	return nil
}

type certificatesService struct {
	store   *store
	manager sslcert.Manager
}

func (s *certificatesService) CertificatesCreate(ctx context.Context, cert *Certificate) (*Certificate, error) {
	id, err := s.manager.Add(cert.AppID, cert.CertificateChain, cert.PrivateKey)
	if err != nil {
		return cert, err
	}

	cert.Name = id
	return s.store.CertificatesCreate(cert)
}

func (s *certificatesService) CertificatesUpdate(ctx context.Context, cert *Certificate) (*Certificate, error) {
	if err := s.manager.Remove(cert.Name); err != nil {
		return cert, err
	}
	id, err := s.manager.Add(cert.AppID, cert.CertificateChain, cert.PrivateKey)
	if err != nil {
		return cert, err
	}

	cert.Name = id
	_, err = s.store.CertificatesUpdate(cert)
	return cert, err
}

func (s *certificatesService) CertificatesDestroy(ctx context.Context, cert *Certificate) error {
	if err := s.manager.Remove(cert.Name); err != nil {
		return err
	}
	return s.store.CertificatesDestroy(cert)
}

func (s *store) CertificatesCreate(cert *Certificate) (*Certificate, error) {
	return certificatesCreate(s.db, cert)
}

func (s *store) CertificatesUpdate(cert *Certificate) (int64, error) {
	return certificatesUpdate(s.db, cert)
}

func (s *store) CertificatesDestroy(cert *Certificate) error {
	return certificatesDestroy(s.db, cert)
}

func (s *store) CertificatesFind(id string) (*Certificate, error) {
	return certificatesFindBy(s.db, "id", id)
}

func (s *store) CertificatesFindByApp(app string) (*Certificate, error) {
	return certificatesFindBy(s.db, "app_id", app)
}

func certificatesCreate(db *db, cert *Certificate) (*Certificate, error) {
	return cert, db.Insert(cert)
}

func certificatesUpdate(db *db, cert *Certificate) (int64, error) {
	return db.Update(cert)
}
func certificatesDestroy(db *db, cert *Certificate) error {
	_, err := db.Delete(cert)
	return err
}

func certificatesFindBy(db *db, field string, value interface{}) (*Certificate, error) {
	var cert Certificate

	if err := findBy(db, &cert, "certificates", field, value); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return &cert, nil
}
