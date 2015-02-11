package units

import (
	"errors"
	"fmt"

	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/releases"
	"github.com/remind101/empire/slugs"
)

var (
	ErrReleaseNotFound     = errors.New("Release not found for app")
	ErrProcessTypeNotFound = errors.New("Process type not found for app")
)

type UnitMap map[slugs.ProcessType]Unit

type Unit struct {
	Release       *releases.Release
	ProcessType   slugs.ProcessType `json:"process_type"`
	InstanceCount int               `json:"instance_count"`
}

func NewUnit(rel *releases.Release, proctype slugs.ProcessType, count int) Unit {
	return Unit{
		Release:       rel,
		ProcessType:   proctype,
		InstanceCount: count,
	}
}

func (u Unit) String() string {
	return fmt.Sprintf("%v.%v release=%v count=%v", u.Release.App.ID, u.ProcessType, u.Release.ID, u.InstanceCount)
}

// Repository is an interface for storing a Unit
type Repository interface {
	Create(*releases.Release) error
	Put(Unit) error
	Delete(Unit) error
	FindByApp(apps.ID) ([]Unit, error)
}

type Service struct {
	Repository
}

func NewService(r Repository) *Service {
	return &Service{Repository: r}
}

// CreateRelease creates a new release for a repo
//
// If existing process definitions exist, they are updated with the new release
// Else a process definition is created for each process type in the release
// with an instance count of 0. The special case `web` will get an initial
// instance count of 1.
//
// Additionally, any process definitions that do not exist in this release's
// process types will be deleted.
func (s *Service) CreateRelease(rel *releases.Release) error {
	// Create the release
	err := s.Repository.Create(rel)
	if err != nil {
		return err
	}

	// Find existing units
	units, err := s.FindByApp(rel.App.ID)
	if err != nil {
		return err
	}

	// Create a unit map
	unitmap := make(map[slugs.ProcessType]Unit)
	for _, u := range units {
		unitmap[u.ProcessType] = u
	}

	// For each process type in new release,
	// update or create a unit
	for pt := range rel.Slug.ProcessTypes {
		var u Unit
		var found bool

		if u, found = unitmap[pt]; found {
			u.Release = rel
		} else {
			count := 0
			if pt == "web" {
				count = 1
			}
			u = NewUnit(rel, pt, count)
		}

		err := s.Repository.Put(u)
		if err != nil {
			return err
		}
	}

	// For each existing process definition
	// If not included in new release's proc types, delete it
	for pt, u := range unitmap {
		if _, found := rel.Slug.ProcessTypes[pt]; !found {
			err := s.Repository.Delete(u)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Scale updates a unit's instance count
func (s *Service) Scale(id apps.ID, proctype slugs.ProcessType, count int) (Unit, error) {
	var u Unit
	var err error

	units, err := s.FindByApp(id)
	if err != nil {
		return u, err
	}

	if len(units) == 0 {
		return u, ErrReleaseNotFound
	}

	rel := units[0].Release
	if _, ok := rel.Slug.ProcessTypes[proctype]; !ok {
		return u, ErrProcessTypeNotFound
	}

	u = NewUnit(rel, proctype, count)
	err = s.Repository.Put(u)
	if err != nil {
		return u, err
	}

	return u, nil
}

func (s *Service) FindByApp(id apps.ID) ([]Unit, error) {
	return s.Repository.FindByApp(id)
}

func (s *Service) Delete(id apps.ID, proctype slugs.ProcessType) error {
	units, err := s.FindByApp(id)
	if err != nil {
		return err
	}

	for _, u := range units {
		if string(proctype) == "" || proctype == u.ProcessType {
			err := s.Repository.Delete(u)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
