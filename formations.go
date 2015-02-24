package empire

import (
	"errors"

	"github.com/remind101/empire/formations"
)

var (
	// Scaling errors
	ErrNoFormation        = errors.New("no process formation for app")
	ErrInvalidProcessType = errors.New("no matching process type")
)

// FormationsService represents a service for configuring the apps process
// formation.
type FormationsService interface {
	formations.Repository
}

// formationsService is a service for managing process formations for apps.
type formationsService struct {
	formations.Repository
}

// NewFormationsService returns a new Service instance.
func NewFormationsService(options Options) (FormationsService, error) {
	return &formationsService{
		Repository: formations.NewRepository(),
	}, nil
}
