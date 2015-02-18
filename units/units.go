package units

import (
	"errors"
	"fmt"

	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/formations"
	"github.com/remind101/empire/releases"
)

var (
	ErrReleaseNotFound     = errors.New("Release not found for app")
	ErrProcessTypeNotFound = errors.New("Process type not found for app")
)

type Name string

type UnitMap map[Name]Unit

type Unit struct {
	Release       *releases.Release
	ProcessType   formations.ProcessType `json:"process_type"`
	InstanceCount int                    `json:"instance_count"`
}

func NewUnit(rel *releases.Release, proctype formations.ProcessType, count int) Unit {
	return Unit{
		Release:       rel,
		ProcessType:   proctype,
		InstanceCount: count,
	}
}

func (u Unit) Name() Name {
	return GenName(u.Release.App.ID, u.ProcessType)
}

func (u Unit) String() string {
	return fmt.Sprintf("%v release=%v count=%v", u.Name(), u.Release.ID, u.InstanceCount)
}

func GenName(id apps.ID, pt formations.ProcessType) Name {
	return Name(fmt.Sprintf("%v.%v", id, pt))
}

// Repository is an interface for storing a Unit
type Repository interface {
	FindByName(Name) (Unit, bool, error)
	FindByApp(apps.ID) (UnitMap, error)
	FindAll() (UnitMap, error)
	Put(Unit) error
	Delete(Unit) error
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
	// Find existing units
	unitmap, err := s.FindByApp(rel.App.ID)
	if err != nil {
		return err
	}

	// For each process type in new release,
	// update or create a unit
	for pt := range rel.Slug.ProcessTypes {
		var u Unit
		var found bool

		n := GenName(rel.App.ID, pt)

		if u, found = unitmap[n]; found {
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
	for _, u := range unitmap {
		if _, found := rel.Slug.ProcessTypes[u.ProcessType]; !found {
			err := s.Repository.Delete(u)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Scale updates a unit's instance count
func (s *Service) Scale(n Name, count int) (Unit, error) {
	var u Unit
	var err error

	u, found, err := s.Repository.FindByName(n)
	if err != nil {
		return u, err
	}
	if !found {
		return u, ErrProcessTypeNotFound
	}

	u.InstanceCount = count
	err = s.Repository.Put(u)
	if err != nil {
		return u, err
	}

	return u, nil
}

func (s *Service) FindByApp(id apps.ID) (UnitMap, error) {
	return s.Repository.FindByApp(id)
}

func (s *Service) Delete(id apps.ID, proctype formations.ProcessType) error {
	unitmap, err := s.FindByApp(id)
	if err != nil {
		return err
	}

	for _, u := range unitmap {
		if string(proctype) == "" || proctype == u.ProcessType {
			err := s.Repository.Delete(u)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
