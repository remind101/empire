package empire

import (
	"database/sql"
	"time"

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

type CertificateManager interface {
	Upload(name string, crt string, key string) (id string, err error)
	MetaData(id string) (data map[string]string, err error)
	Remove(id string) (err error)
}

type fakeCertificateManager struct{}

func (m *fakeCertificateManager) Upload(name string, crt string, key string) (string, error) {
	return "fake", nil
}

func (m *fakeCertificateManager) Remove(id string) error {
	return nil
}

func (m *fakeCertificateManager) MetaData(id string) (map[string]string, error) {
	return map[string]string{}, nil
}

type certificatesService struct {
	store   *store
	manager CertificateManager
}

func (s *certificatesService) CertificatesCreate(ctx context.Context, cert *Certificate) (*Certificate, error) {
	id, err := s.manager.Upload(cert.AppID, cert.CertificateChain, cert.PrivateKey)
	if err != nil {
		return cert, err
	}

	cert.Name = id
	return s.store.CertificatesCreate(cert)
}

func (s *certificatesService) CertificatesUpdate(ctx context.Context, cert *Certificate) (*Certificate, error) {
	_, err := s.store.CertificatesUpdate(cert)
	return cert, err
}

func (s *certificatesService) CertificatesDestroy(ctx context.Context, cert *Certificate) error {
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
