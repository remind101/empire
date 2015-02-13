package procs

import (
	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/releases"
	"github.com/remind101/empire/slugs"
	"github.com/remind101/empire/units"
)

type Name string

type Process struct {
	ID          int
	AppID       apps.ID
	ReleaseID   releases.ID
	ProcessType slugs.ProcessType
	Minion      string
}

type ProcessMap map[Name][]Process

type Scheduler struct {
	UnitsService *units.Service
	Repository
}

func (s *Scheduler) ScheduleUnits() error {
	unitsmap, err := s.UnitsService.FindAll()
	if err != nil {
		return err
	}
}

type Repository interface {
	FindAll() (ProcessMap, error)
}
