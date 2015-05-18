package empire

import (
	"time"

	"github.com/jinzhu/gorm"
	"github.com/remind101/empire/empire/pkg/sslcert"
	"github.com/remind101/pkg/timex"
	"golang.org/x/net/context"
)

type Certificate struct {
	ID               string
	Name             string
	CertificateChain string
	PrivateKey       string `sql:"-"`
	CreatedAt        *time.Time
	UpdatedAt        *time.Time

	AppID string
	App   *App
}

// PreInsert implements a pre insert hook for the db interface
func (c *Certificate) BeforeCreate() error {
	t := timex.Now()
	c.CreatedAt = &t
	c.UpdatedAt = &t
	return nil
}

// PreUpdate implements a pre insert hook for the db interface
func (c *Certificate) BeforeUpdate() error {
	t := timex.Now()
	c.UpdatedAt = &t
	return nil
}

type certificatesService struct {
	store    *store
	manager  sslcert.Manager
	releaser *releaser
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
	return cert, s.store.CertificatesUpdate(cert)
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

func (s *store) CertificatesUpdate(cert *Certificate) error {
	return certificatesUpdate(s.db, cert)
}

func (s *store) CertificatesDestroy(cert *Certificate) error {
	return certificatesDestroy(s.db, cert)
}

func (s *store) CertificatesFind(scope func(*gorm.DB) *gorm.DB) (*Certificate, error) {
	var cert Certificate
	if err := s.db.Scopes(scope).First(&cert).Error; err != nil {
		if err == gorm.RecordNotFound {
			return nil, nil
		}

		return nil, err
	}
	return &cert, nil
}

func CertificateID(id string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", id)
	}
}

func CertificateApp(app *App) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("app_id = ?", app.ID)
	}
}

func certificatesCreate(db *gorm.DB, cert *Certificate) (*Certificate, error) {
	return cert, db.Create(cert).Error
}

func certificatesUpdate(db *gorm.DB, cert *Certificate) error {
	return db.Save(cert).Error
}
func certificatesDestroy(db *gorm.DB, cert *Certificate) error {
	return db.Delete(cert).Error
}
