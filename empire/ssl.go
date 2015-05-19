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
	id, err := s.manager.Add(certName(cert), cert.CertificateChain, cert.PrivateKey)
	if err != nil {
		return cert, err
	}

	cert.Name = id
	return s.store.CertificatesCreate(cert)
}

func (s *certificatesService) CertificatesUpdate(ctx context.Context, cert *Certificate) (*Certificate, error) {
	if err := s.manager.Remove(certName(cert)); err != nil {
		return cert, err
	}
	id, err := s.manager.Add(certName(cert), cert.CertificateChain, cert.PrivateKey)
	if err != nil {
		return cert, err
	}

	cert.Name = id
	return cert, s.store.CertificatesUpdate(cert)
}

func (s *certificatesService) CertificatesDestroy(ctx context.Context, cert *Certificate) error {
	if err := s.manager.Remove(certName(cert)); err != nil {
		return err
	}
	return s.store.CertificatesDestroy(cert)
}

// certName is the cert name we pass to our cert manager.
func certName(cert *Certificate) string {
	return cert.AppID
}

// CertificatesQuery is a Scope implementation for common things to filter
// certificates by.
type CertificatesQuery struct {
	// If provided, finds the certificate with the given id.
	ID *string

	// If provided, filters certificates belong to the given app.
	App *App
}

// Scope implements the Scope interface.
func (q CertificatesQuery) Scope(db *gorm.DB) *gorm.DB {
	var scope ComposedScope

	if q.ID != nil {
		scope = append(scope, ID(*q.ID))
	}

	if q.App != nil {
		scope = append(scope, ForApp(q.App))
	}

	return scope.Scope(db)
}

// CertificatesFirst returns the first matching certificate.
func (s *store) CertificatesFirst(scope Scope) (*Certificate, error) {
	var cert Certificate
	return &cert, s.First(scope, &cert)
}

// CertificatesCreate persists the certificate.
func (s *store) CertificatesCreate(cert *Certificate) (*Certificate, error) {
	return certificatesCreate(s.db, cert)
}

// CertificatesUpdate updates the certificate.
func (s *store) CertificatesUpdate(cert *Certificate) error {
	return certificatesUpdate(s.db, cert)
}

// CertificatesDestroy destroys the certificate.
func (s *store) CertificatesDestroy(cert *Certificate) error {
	return certificatesDestroy(s.db, cert)
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
